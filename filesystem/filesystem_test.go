package filesystem

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/hanwen/go-fuse/v2/fs"

	"github.com/joshkunz/fakelib/library"
)

const (
	goldFilePath = "testdata/gold.mp3"
)

func loadLibrary(t *testing.T) *library.Library {
	t.Helper()

	gold, err := os.Open(goldFilePath)
	if err != nil {
		t.Fatalf("Failed to load %q: %v", goldFilePath, err)
	}
	defer gold.Close()

	lib, err := library.New(gold)
	if err != nil {
		t.Fatalf("Failed to create new library: %v", err)
	}
	return lib
}

func mount(t *testing.T, lib *library.Library) (dir string, cleanup func()) {
	t.Helper()

	d, err := ioutil.TempDir("", "fakelib-filesystem")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	srv, err := Mount(lib, d, &fs.Options{})
	if err != nil {
		t.Fatalf("Failed to mount FUSE server at %q: %v", d, err)
	}
	return d, func() {
		if err := srv.Unmount(); err != nil {
			t.Errorf("Failed to unmount: %v", err)
		}
		if err := os.Remove(d); err != nil {
			t.Errorf("Failed to remove mount dir %q: %v", d, err)
		}
	}
}

// Test that we can mount the filesystem, stat a file, and unmount.
func TestBasic(t *testing.T) {
	dir, cleanup := mount(t, loadLibrary(t))

	// The library is now mounted, we should be able to stat `A/A/A.mp3`.
	if _, err := os.Stat(dir + "/A/A/A.mp3"); err != nil {
		t.Errorf("Failed to stat A/A/A.mp3: %v", err)
	}

	// This unmounts the test library filesystem.
	cleanup()

	// The library is no longer mounted, stat of A/A/A.mp3 should fail.
	if _, err := os.Stat(dir + "/A/A/A.mp3"); err == nil { // note: == nil
		t.Errorf("Stat of A/A/A.mp3 succeeded after unmount, should fail.")
	}
}

// MPD (musicpd.org) has a recursive-folder detection algorithm based on file
// Inode number. Unfortunately, it representes inodes as C `unsigned` integers
// which are typically 32 bits. This is problematic for us because go-fuse by
// default generates very large (~63 bits) inode numbers for each file.
// Truncating to 32 bits means that some files/directories appear to have the
// same inode numbers as one of their parents, and MPD's recursive folder
// detection is triggered.
//
// This test verifies that even for large libraries we generate relatively small
// (within 32-bits) inode numbers for the files and directories. It also
// verifies that we always generate unique Inodes.
func TestSmallUniqueInodes(t *testing.T) {
	lib := loadLibrary(t)
	// Make the library pretty large, ~20k
	lib.Tracks = 20_000

	dir, cleanup := mount(t, lib)
	defer cleanup()

	// The largest Inode we will support, something a good bit larger than our
	// our library size.
	maxInode := uint64(lib.Tracks * 2)
	allInodes := make(map[uint64]string)

	var failures int
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("Item at path %q did not have Stat_t file attribute", path)
		}

		var hadFailure bool

		if stat.Ino > maxInode {
			t.Errorf("Item %q had inode %d, want < %d", path, stat.Ino, maxInode)
			hadFailure = true
		}

		// filepath.Walk appears to traverse some paths twice, which
		// triggers this condition. Also check path here in-case this
		// happens.
		if ePath, exists := allInodes[stat.Ino]; exists && ePath != path {
			t.Errorf("Item %q had duplicate inode %d at %q", path, stat.Ino, ePath)
			hadFailure = true
		}

		allInodes[stat.Ino] = path

		if hadFailure {
			failures++
			if failures > 100 {
				return errors.New("too many errors occured, aborting")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Error while walking mount: %v, want nil", err)
	}
}
