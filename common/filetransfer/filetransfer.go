// Package filetransfer contains helpers shared by the inner and outer file
// transfer services
package filetransfer

import (
	"crypto/sha256"
	"dd-nats/common/types"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SafeJoin joins a base directory with a relative path received from another
// system (in forward slash format) and makes sure the result is inside the base
// directory. Absolute paths and paths escaping the base directory are rejected.
func SafeJoin(base string, elems ...string) (string, error) {
	for i := range elems {
		elems[i] = strings.ReplaceAll(elems[i], "\\", "/")
	}

	rel := path.Join(elems...) // also cleans the path
	if strings.Contains(rel, ":") {
		return "", fmt.Errorf("drive letters and ':' are not allowed in path: %s", rel)
	}

	// Leading slashes are relative to the base directory
	rel = path.Clean(strings.TrimLeft(rel, "/"))
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("path escapes base directory: %s", rel)
	}

	return filepath.Join(base, filepath.FromSlash(rel)), nil
}

// HashFile calculates the SHA256 hash of a file
func HashFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

type hashEntry struct {
	size    int64
	modtime time.Time
	hash    []byte
}

// ManifestBuilder builds file manifests for a directory tree and caches the
// file hashes so unchanged files are not hashed again
type ManifestBuilder struct {
	lock     sync.RWMutex
	cache    map[string]hashEntry
	manifest *types.FileManifest
}

func NewManifestBuilder() *ManifestBuilder {
	return &ManifestBuilder{cache: make(map[string]hashEntry), manifest: &types.FileManifest{Files: []types.FileInfo{}}}
}

// Manifest returns the last built manifest, never nil
func (mb *ManifestBuilder) Manifest() types.FileManifest {
	mb.lock.RLock()
	defer mb.lock.RUnlock()
	return *mb.manifest
}

// Build creates a new manifest for the directory 'root'. Errors for single
// files are reported through onError and the file is left out of the manifest.
func (mb *ManifestBuilder) Build(root string, onError func(err error)) *types.FileManifest {
	manifest := &types.FileManifest{LastUpdate: time.Now().UTC(), Files: []types.FileInfo{}}
	seen := make(map[string]bool)

	filepath.Walk(root, func(fullname string, fi os.FileInfo, err error) error {
		if err != nil {
			if onError != nil {
				onError(err)
			}
			return nil
		}

		if fi.IsDir() {
			return nil
		}

		reldir, _ := filepath.Rel(root, filepath.Dir(fullname))
		dirname := "/" + strings.TrimPrefix(filepath.ToSlash(reldir), ".")
		dirname = path.Clean(dirname)

		mb.lock.RLock()
		entry, ok := mb.cache[fullname]
		mb.lock.RUnlock()

		if !ok || entry.size != fi.Size() || !entry.modtime.Equal(fi.ModTime()) {
			hash, err := HashFile(fullname)
			if err != nil {
				if onError != nil {
					onError(err)
				}
				return nil
			}

			entry = hashEntry{size: fi.Size(), modtime: fi.ModTime(), hash: hash}
			mb.lock.Lock()
			mb.cache[fullname] = entry
			mb.lock.Unlock()
		}

		seen[fullname] = true
		info := types.FileInfo{Name: fi.Name(), Path: dirname, Size: int(fi.Size()), Date: fi.ModTime(), Hash: entry.hash}
		manifest.Files = append(manifest.Files, info)
		return nil
	})

	mb.lock.Lock()
	for name := range mb.cache {
		if !seen[name] {
			delete(mb.cache, name)
		}
	}
	mb.manifest = manifest
	mb.lock.Unlock()

	return manifest
}

// PruneOlderThan removes files in the directory tree 'root' that are older
// than maxAge, and returns the number of removed files
func PruneOlderThan(root string, maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, nil
	}

	count := 0
	err := filepath.Walk(root, func(fullname string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}

		if time.Since(fi.ModTime()) > maxAge {
			if err := os.Remove(fullname); err == nil {
				count++
			}
		}

		return nil
	})

	return count, err
}
