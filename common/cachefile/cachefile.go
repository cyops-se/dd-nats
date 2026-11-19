// Package cachefile reads the gzipped process data cache files produced by
// dd-nats-cache (JSON arrays of data points)
package cachefile

import (
	"compress/gzip"
	"dd-nats/common/types"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Subdirectories of the cache directory used to keep track of processed files
const (
	DoneDir   = "done"
	FailedDir = "failed"
)

// ReadDataPoints reads all data points in a cache file and calls fn for each
// of them. Files with several concatenated gzip members and/or several JSON
// arrays (created by older versions when the cache service was restarted)
// are handled. Data points read before an error are still delivered.
func ReadDataPoints(filename string, fn func(dp types.DataPoint)) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return 0, fmt.Errorf("not a valid gzip file: %s", err.Error())
	}
	defer gzr.Close()

	count := 0
	decoder := json.NewDecoder(gzr)
	for {
		var datapoints types.DataPoints
		err := decoder.Decode(&datapoints)
		if errors.Is(err, io.EOF) {
			return count, nil
		}

		if err != nil {
			return count, fmt.Errorf("failed to decode content after %d data points: %s", count, err.Error())
		}

		for _, dp := range datapoints {
			fn(dp)
			count++
		}
	}
}

// ProcessDirectory processes all cache files (*.gz) in the directory tree 'root'
// and moves each processed file to root/done (or root/failed if it couldn't
// be read), keeping the relative path. The done and failed directories are
// not processed. Errors for single files are reported through onError and
// processing continues with the next file.
func ProcessDirectory(root string, fn func(dp types.DataPoint), onError func(filename string, err error)) (files int, points int, err error) {
	skip := map[string]bool{
		filepath.Join(root, DoneDir):      true,
		filepath.Join(root, FailedDir):    true,
		filepath.Join(root, "processing"): true,
	}

	err = filepath.Walk(root, func(fullname string, info os.FileInfo, err error) error {
		if err != nil {
			if onError != nil {
				onError(fullname, err)
			}
			return nil
		}

		if info.IsDir() {
			if skip[fullname] {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.EqualFold(filepath.Ext(info.Name()), ".gz") {
			return nil
		}

		n, rerr := ReadDataPoints(fullname, fn)
		points += n
		files++

		todir := DoneDir
		if rerr != nil {
			todir = FailedDir
			if onError != nil {
				onError(fullname, rerr)
			}
		}

		if merr := moveFile(root, fullname, todir); merr != nil && onError != nil {
			onError(fullname, merr)
		}

		return nil
	})

	return files, points, err
}

func moveFile(root string, fullname string, todir string) error {
	rel, err := filepath.Rel(root, fullname)
	if err != nil {
		return err
	}

	target := filepath.Join(root, todir, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	return os.Rename(fullname, target)
}
