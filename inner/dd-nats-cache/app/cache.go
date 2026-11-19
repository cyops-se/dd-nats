package app

import (
	"bufio"
	"compress/gzip"
	"dd-nats/common/ddsvc"
	"dd-nats/common/types"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	cacheDir       = "cache"
	rotateInterval = 5 * time.Minute // New file every 5 minutes
	flushInterval  = 10 * time.Second
)

var lsvc *ddsvc.DdUsvc

// The current cache file, protected by writerLock
var writerLock sync.Mutex
var file *os.File
var gzw *gzip.Writer
var fw *bufio.Writer
var firstWrite bool
var currentBucket time.Time

type CacheItem struct {
	Filename string    `json:"filename"`
	Time     time.Time `json:"time"`
	Size     int64     `json:"size"`
}

type CacheInfo struct {
	Size      int64       `json:"size"`
	Count     int         `json:"count"`
	FirstTime time.Time   `json:"firsttime"`
	LastTime  time.Time   `json:"lasttime"`
	Items     []CacheItem `json:"items"`
}

var cacheInfo CacheInfo
var cacheMutex sync.Mutex

func InitCache(svc *ddsvc.DdUsvc) {
	lsvc = svc
	lsvc.GetInt("cache.retention", 7)

	svc.Subscribe("process.actual", processMessages)
	svc.OnStop(CloseCache)

	go flushCache()
	go pruneCache()
}

func RefreshCache() {
	refreshCache()
}

// CloseCache closes the current cache file (if any) properly
func CloseCache() {
	writerLock.Lock()
	defer writerLock.Unlock()
	closeFile()
}

// Must be called with writerLock held
func closeFile() {
	if fw == nil {
		return
	}

	if firstWrite {
		fw.Write([]byte("[")) // no data written, make it a valid (empty) JSON array
	}

	fw.Write([]byte("]"))
	if err := fw.Flush(); err != nil {
		lsvc.Error("Cache", "Failed to flush cache file: %s", err.Error())
	}

	if err := gzw.Close(); err != nil {
		lsvc.Error("Cache", "Failed to close gzip stream: %s", err.Error())
	}

	if err := file.Close(); err != nil {
		lsvc.Error("Cache", "Failed to close cache file: %s", err.Error())
	}

	fw, gzw, file = nil, nil, nil
}

// GetCacheInfo returns a snapshot of the cache content
func GetCacheInfo() CacheInfo {
	refreshCache()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	info := cacheInfo
	info.Items = append([]CacheItem(nil), cacheInfo.Items...)
	if info.Count > 0 {
		info.FirstTime = info.Items[0].Time
		info.LastTime = info.Items[info.Count-1].Time
	}

	return info
}

// ResendCacheItems copies the requested cache files (if they exist in the
// cache) to the file transfer outgoing directory and returns the number of
// files successfully queued for transfer
func ResendCacheItems(items []CacheItem) int {
	info := GetCacheInfo()
	known := make(map[string]bool, len(info.Items))
	for _, fi := range info.Items {
		known[fi.Filename] = true
	}

	count := 0
	for _, item := range items {
		if known[item.Filename] {
			count += resendCacheItem(item)
		} else {
			lsvc.Error("Cache", "Requested cache file not found in cache: %s", item.Filename)
		}
	}

	return count
}

func SendFullCache() error {
	// Just copy the files to the outgoing file transfer directory
	return copyDir(cacheDir, filepath.Join("outgoing", "new"))
}

func copyDir(source, destination string) error {
	err := filepath.Walk(source, func(fullname string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, fullname)
		if err != nil || relPath == "." {
			return err
		}

		if info.IsDir() {
			return os.MkdirAll(filepath.Join(destination, relPath), 0755)
		}

		return copyFile(fullname, filepath.Join(destination, relPath))
	})

	if err != nil {
		lsvc.Error("Cache", "copy command failed: %s", err.Error())
	}

	return err
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	dir := filepath.Dir(dst)
	lsvc.Trace("Cache", "Creating directory %s", dir)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Write to a temporary name first so the file transfer service doesn't pick up a partial file
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return
	}

	if _, err = io.Copy(out, in); err == nil {
		err = out.Sync()
	}

	if cerr := out.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		os.Remove(tmp)
		return
	}

	return os.Rename(tmp, dst)
}

func resendCacheItem(item CacheItem) int {
	// First put it on the file transfer directory
	newFilename := filepath.Join("outgoing", "new", filepath.FromSlash(strings.ReplaceAll(item.Filename, "\\", "/")))
	if err := copyFile(item.Filename, newFilename); err != nil {
		lsvc.Error("Cache", "Failed to copy file %s to %s, error: %s", item.Filename, newFilename, err.Error())
		return 0
	}

	return 1
}

func getTimeFromFilename(filename string) time.Time {
	var year, day, hour, minute int
	var month time.Month
	fmt.Sscanf(filename, "dd_%d_%02d_%02d-%02d_%02d", &year, &month, &day, &hour, &minute)
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func refreshCache() {
	var items []CacheItem
	var size int64

	err := filepath.Walk(cacheDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip files that can't be read, e.g. removed while walking
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json.gz") {
			items = append(items, CacheItem{Filename: p, Time: getTimeFromFilename(info.Name()), Size: info.Size()})
			size += info.Size()
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		lsvc.Error("Cache", "Filewalk error: %s", err.Error())
	}

	cacheMutex.Lock()
	cacheInfo.Items = items
	cacheInfo.Size = size
	cacheInfo.Count = len(items)
	cacheMutex.Unlock()
}

func processMessages(subject string, responseTopic string, data []byte) error {
	var msg types.DataPoint
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal data point: %s", err.Error())
	}

	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	writerLock.Lock()
	defer writerLock.Unlock()

	bucket := time.Now().UTC().Truncate(rotateInterval)
	if fw == nil || !bucket.Equal(currentBucket) {
		closeFile()
		createFile(bucket)
	}

	if fw == nil {
		return nil // createFile already logged the error
	}

	if firstWrite {
		fw.Write([]byte("["))
	} else {
		fw.Write([]byte(","))
	}

	fw.Write(content)
	firstWrite = false

	return nil
}

// Must be called with writerLock held
func createFile(bucket time.Time) {
	currentBucket = bucket
	dirpath := filepath.Join(cacheDir, fmt.Sprintf("%d", bucket.Year()), fmt.Sprintf("%02d", bucket.Month()), fmt.Sprintf("%02d", bucket.Day()))
	basename := fmt.Sprintf("dd_%d_%02d_%02d-%02d_%02d", bucket.Year(), bucket.Month(), bucket.Day(), bucket.Hour(), bucket.Minute())

	if err := os.MkdirAll(dirpath, 0755); err != nil {
		lsvc.Error("Cache", "Failed to create cache directory %s: %s", dirpath, err.Error())
		return
	}

	// Never append to an existing file (e.g. after a restart within the same
	// interval), that would create an invalid JSON document. Use a new name instead.
	var f *os.File
	var err error
	for n := 0; n < 100; n++ {
		name := basename + ".json.gz"
		if n > 0 {
			name = fmt.Sprintf("%s-%d.json.gz", basename, n)
		}

		f, err = os.OpenFile(filepath.Join(dirpath, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil || !os.IsExist(err) {
			break
		}
	}

	if err != nil {
		lsvc.Error("Cache", "Failed to create cache file in %s: %s", dirpath, err.Error())
		return
	}

	file = f
	gzw = gzip.NewWriter(file)
	fw = bufio.NewWriter(gzw)
	firstWrite = true
}

// Periodically flush buffered data so that at most a few seconds of data is
// lost if the process dies (the gzip stream can be recovered up to the last flush)
func flushCache() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for range ticker.C {
		writerLock.Lock()
		if fw != nil {
			if err := fw.Flush(); err == nil {
				gzw.Flush()
			}
		}

		// Close the file if no data arrived during the whole interval
		if fw != nil && !time.Now().UTC().Truncate(rotateInterval).Equal(currentBucket) {
			closeFile()
		}
		writerLock.Unlock()
	}
}

func pruneCache() {
	// Check once an hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		days := lsvc.GetInt("cache.retention", 7)
		if days < 1 {
			days = 7
		}

		refreshCache()

		cacheMutex.Lock()
		items := append([]CacheItem(nil), cacheInfo.Items...)
		cacheMutex.Unlock()

		utc := time.Now().UTC()
		count := 0
		for _, item := range items {
			if utc.Sub(item.Time) > time.Duration(days)*24*time.Hour {
				if err := os.Remove(item.Filename); err == nil {
					count++
				}
			}
		}

		if count > 0 {
			lsvc.Trace("Cache pruned", "%d files pruned from cache", count)
			refreshCache()
		}

		<-ticker.C
	}
}
