package cachefile

import (
	"compress/gzip"
	"dd-nats/common/types"
	"os"
	"path/filepath"
	"testing"
)

func writeGz(t *testing.T, filename string, members ...string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(filename), 0755)
	f, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	for _, m := range members {
		w := gzip.NewWriter(f)
		w.Write([]byte(m))
		w.Close()
	}
}

func TestReadDataPoints(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		members []string
		count   int
		fail    bool
	}{
		{"single", []string{`[{"n":"a","v":1},{"n":"b","v":2}]`}, 2, false},
		{"empty", []string{`[]`}, 0, false},
		{"concatenated", []string{`[{"n":"a","v":1}]`, `[{"n":"b","v":2},{"n":"c","v":3}]`}, 3, false},
		{"truncated", []string{`[{"n":"a","v":1},{"n":"b"`}, 0, true},
		{"only bracket", []string{`]`}, 0, true},
	}

	for _, c := range cases {
		filename := filepath.Join(dir, c.name+".json.gz")
		writeGz(t, filename, c.members...)

		count := 0
		n, err := ReadDataPoints(filename, func(dp types.DataPoint) { count++ })
		if (err != nil) != c.fail {
			t.Errorf("%s: unexpected error state: %v", c.name, err)
		}
		if n != c.count || count != c.count {
			t.Errorf("%s: expected %d points, got %d (%d)", c.name, c.count, n, count)
		}
	}
}

func TestProcessDirectory(t *testing.T) {
	root := t.TempDir()
	writeGz(t, filepath.Join(root, "2024", "01", "01", "good.json.gz"), `[{"n":"a","v":1}]`)
	writeGz(t, filepath.Join(root, "2024", "01", "01", "bad.json.gz"), `[{"n":`)
	os.WriteFile(filepath.Join(root, "notgz.txt"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(root, "corrupt.gz"), []byte("not gzip"), 0644)

	var errs int
	files, points, err := ProcessDirectory(root, func(dp types.DataPoint) {}, func(string, error) { errs++ })
	if err != nil {
		t.Fatal(err)
	}

	if files != 3 || points != 1 || errs != 2 {
		t.Errorf("files %d, points %d, errors %d", files, points, errs)
	}

	if _, err := os.Stat(filepath.Join(root, DoneDir, "2024", "01", "01", "good.json.gz")); err != nil {
		t.Errorf("good file not moved to done: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, FailedDir, "corrupt.gz")); err != nil {
		t.Errorf("corrupt file not moved to failed: %v", err)
	}

	// Second run should not process anything
	files, _, _ = ProcessDirectory(root, func(dp types.DataPoint) {}, nil)
	if files != 0 {
		t.Errorf("expected no files on second run, got %d", files)
	}
}

func TestProcessDirectoryMissing(t *testing.T) {
	errors := 0
	files, points, err := ProcessDirectory(filepath.Join(t.TempDir(), "does-not-exist"), func(dp types.DataPoint) {}, func(string, error) { errors++ })
	if err != nil || files != 0 || points != 0 || errors != 0 {
		t.Fatalf("got files %d points %d errors %d err %v, expected nothing", files, points, errors, err)
	}
}
