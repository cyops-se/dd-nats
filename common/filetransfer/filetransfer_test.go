package filetransfer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	base := filepath.Join("base", "dir")

	ok := map[string][]string{
		filepath.Join(base, "file.txt"):        {"/", "file.txt"},
		filepath.Join(base, "sub", "file.txt"): {"/sub", "file.txt"},
		filepath.Join(base, "x", "file.txt"):   {"sub/../x", "file.txt"},
		filepath.Join(base, "file2.txt"):       {".", "file2.txt"},
		filepath.Join(base, "a", "b"):          {"a\\b"},
		filepath.Join(base, "c"):               {"/../c"}, // anchored at base, cannot escape
		filepath.Join(base, "etc", "passwd"):   {"/", "../../etc/passwd"},
	}

	for expected, elems := range ok {
		got, err := SafeJoin(base, elems...)
		if err != nil || got != expected {
			t.Errorf("SafeJoin(%v) = %s, %v, expected %s", elems, got, err, expected)
		}
	}

	bad := [][]string{
		{"..", "file"},
		{"sub", "../../etc/passwd"},
		{"sub", "..\\..\\windows\\x"},
		{"C:\\Windows", "x"},
		{"/", "file:stream"},
	}

	for _, elems := range bad {
		if got, err := SafeJoin(base, elems...); err == nil {
			t.Errorf("SafeJoin(%v) = %s, expected error", elems, got)
		}
	}
}

func TestManifestBuilder(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "sub"), 0755)
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("b"), 0644)

	mb := NewManifestBuilder()
	m := mb.Build(root, func(err error) { t.Error(err) })
	if len(m.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(m.Files))
	}

	paths := map[string]string{}
	for _, f := range m.Files {
		paths[f.Name] = f.Path
		if len(f.Hash) != 32 {
			t.Errorf("unexpected hash length for %s", f.Name)
		}
	}

	if paths["a.txt"] != "/" || paths["b.txt"] != "/sub" {
		t.Errorf("unexpected paths: %v", paths)
	}

	os.Remove(filepath.Join(root, "a.txt"))
	m = mb.Build(root, nil)
	if len(m.Files) != 1 || len(mb.cache) != 1 {
		t.Errorf("expected 1 file and 1 cache entry, got %d and %d", len(m.Files), len(mb.cache))
	}
}
