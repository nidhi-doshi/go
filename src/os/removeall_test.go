// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os_test

import (
	"fmt"
	"os"
	. "os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRemoveAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestRemoveAll-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tmpDir)

	if err := RemoveAll(""); err != nil {
		t.Errorf("RemoveAll(\"\"): %v; want nil", err)
	}

	file := filepath.Join(tmpDir, "file")
	path := filepath.Join(tmpDir, "_TestRemoveAll_")
	fpath := filepath.Join(path, "file")
	dpath := filepath.Join(path, "dir")

	// Make a regular file and remove
	fd, err := Create(file)
	if err != nil {
		t.Fatalf("create %q: %s", file, err)
	}
	fd.Close()
	if err = RemoveAll(file); err != nil {
		t.Fatalf("RemoveAll %q (first): %s", file, err)
	}
	if _, err = Lstat(file); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (first)", file)
	}

	// Make directory with 1 file and remove.
	if err := MkdirAll(path, 0777); err != nil {
		t.Fatalf("MkdirAll %q: %s", path, err)
	}
	fd, err = Create(fpath)
	if err != nil {
		t.Fatalf("create %q: %s", fpath, err)
	}
	fd.Close()
	if err = RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q (second): %s", path, err)
	}
	if _, err = Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (second)", path)
	}

	// Make directory with file and subdirectory and remove.
	if err = MkdirAll(dpath, 0777); err != nil {
		t.Fatalf("MkdirAll %q: %s", dpath, err)
	}
	fd, err = Create(fpath)
	if err != nil {
		t.Fatalf("create %q: %s", fpath, err)
	}
	fd.Close()
	fd, err = Create(dpath + "/file")
	if err != nil {
		t.Fatalf("create %q: %s", fpath, err)
	}
	fd.Close()
	if err = RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q (third): %s", path, err)
	}
	if _, err := Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (third)", path)
	}

	// Chmod is not supported under Windows and test fails as root.
	if runtime.GOOS != "windows" && Getuid() != 0 {
		// Make directory with file and subdirectory and trigger error.
		if err = MkdirAll(dpath, 0777); err != nil {
			t.Fatalf("MkdirAll %q: %s", dpath, err)
		}

		for _, s := range []string{fpath, dpath + "/file1", path + "/zzz"} {
			fd, err = Create(s)
			if err != nil {
				t.Fatalf("create %q: %s", s, err)
			}
			fd.Close()
		}
		if err = Chmod(dpath, 0); err != nil {
			t.Fatalf("Chmod %q 0: %s", dpath, err)
		}

		// No error checking here: either RemoveAll
		// will or won't be able to remove dpath;
		// either way we want to see if it removes fpath
		// and path/zzz. Reasons why RemoveAll might
		// succeed in removing dpath as well include:
		//	* running as root
		//	* running on a file system without permissions (FAT)
		RemoveAll(path)
		Chmod(dpath, 0777)

		for _, s := range []string{fpath, path + "/zzz"} {
			if _, err = Lstat(s); err == nil {
				t.Fatalf("Lstat %q succeeded after partial RemoveAll", s)
			}
		}
	}
	if err = RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q after partial RemoveAll: %s", path, err)
	}
	if _, err = Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll (final)", path)
	}
}

// Test RemoveAll on a large directory.
func TestRemoveAllLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "TestRemoveAll-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "_TestRemoveAllLarge_")

	// Make directory with 1000 files and remove.
	if err := MkdirAll(path, 0777); err != nil {
		t.Fatalf("MkdirAll %q: %s", path, err)
	}
	for i := 0; i < 1000; i++ {
		fpath := fmt.Sprintf("%s/file%d", path, i)
		fd, err := Create(fpath)
		if err != nil {
			t.Fatalf("create %q: %s", fpath, err)
		}
		fd.Close()
	}
	if err := RemoveAll(path); err != nil {
		t.Fatalf("RemoveAll %q: %s", path, err)
	}
	if _, err := Lstat(path); err == nil {
		t.Fatalf("Lstat %q succeeded after RemoveAll", path)
	}
}

// chdir changes the current working directory to the named directory,
// and then restore the original working directory at the end of the test.
func chdir(t *testing.T, dir string) {
	olddir, err := os.Getwd()
	if err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(olddir); err != nil {
			t.Errorf("chdir to original working directory %s: %v", olddir, err)
			os.Exit(1)
		}
	})
}

func TestRemoveAllLongPath(t *testing.T) {
	switch runtime.GOOS {
	case "aix", "darwin", "ios", "dragonfly", "freebsd", "linux", "netbsd", "openbsd", "illumos", "solaris":
		break
	default:
		t.Skip("skipping for not implemented platforms")
	}

	startPath, err := os.MkdirTemp("", "TestRemoveAllLongPath-")
	if err != nil {
		t.Fatalf("Could not create TempDir: %s", err)
	}
	defer RemoveAll(startPath)
	chdir(t, startPath)

	// Removing paths with over 4096 chars commonly fails
	for i := 0; i < 41; i++ {
		name := strings.Repeat("a", 100)

		err = Mkdir(name, 0755)
		if err != nil {
			t.Fatalf("Could not mkdir %s: %s", name, err)
		}

		err = Chdir(name)
		if err != nil {
			t.Fatalf("Could not chdir %s: %s", name, err)
		}
	}

	err = RemoveAll(startPath)
	if err != nil {
		t.Errorf("RemoveAll could not remove long file path %s: %s", startPath, err)
	}
}

func TestRemoveAllDot(t *testing.T) {
	prevDir, err := Getwd()
	if err != nil {
		t.Fatalf("Could not get wd: %s", err)
	}
	tempDir, err := os.MkdirTemp("", "TestRemoveAllDot-")
	if err != nil {
		t.Fatalf("Could not create TempDir: %s", err)
	}
	defer RemoveAll(tempDir)

	err = Chdir(tempDir)
	if err != nil {
		t.Fatalf("Could not chdir to tempdir: %s", err)
	}

	err = RemoveAll(".")
	if err == nil {
		t.Errorf("RemoveAll succeed to remove .")
	}

	err = Chdir(prevDir)
	if err != nil {
		t.Fatalf("Could not chdir %s: %s", prevDir, err)
	}
}

func TestRemoveAllDotDot(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "TestRemoveAllDotDot-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tempDir)

	subdir := filepath.Join(tempDir, "x")
	subsubdir := filepath.Join(subdir, "y")
	if err := MkdirAll(subsubdir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAll(filepath.Join(subsubdir, "..")); err != nil {
		t.Error(err)
	}
	for _, dir := range []string{subsubdir, subdir} {
		if _, err := Stat(dir); err == nil {
			t.Errorf("%s: exists after RemoveAll", dir)
		}
	}
}

// Issue #29178.
func TestRemoveReadOnlyDir(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "TestRemoveReadOnlyDir-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tempDir)

	subdir := filepath.Join(tempDir, "x")
	if err := Mkdir(subdir, 0); err != nil {
		t.Fatal(err)
	}

	// If an error occurs make it more likely that removing the
	// temporary directory will succeed.
	defer Chmod(subdir, 0777)

	if err := RemoveAll(subdir); err != nil {
		t.Fatal(err)
	}

	if _, err := Stat(subdir); err == nil {
		t.Error("subdirectory was not removed")
	}
}

// Issue #29983.
func TestRemoveAllButReadOnlyAndPathError(t *testing.T) {
	switch runtime.GOOS {
	case "js", "windows":
		t.Skipf("skipping test on %s", runtime.GOOS)
	}

	if Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	t.Parallel()

	tempDir, err := os.MkdirTemp("", "TestRemoveAllButReadOnly-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tempDir)

	dirs := []string{
		"a",
		"a/x",
		"a/x/1",
		"b",
		"b/y",
		"b/y/2",
		"c",
		"c/z",
		"c/z/3",
	}
	readonly := []string{
		"b",
	}
	inReadonly := func(d string) bool {
		for _, ro := range readonly {
			if d == ro {
				return true
			}
			dd, _ := filepath.Split(d)
			if filepath.Clean(dd) == ro {
				return true
			}
		}
		return false
	}

	for _, dir := range dirs {
		if err := Mkdir(filepath.Join(tempDir, dir), 0777); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range readonly {
		d := filepath.Join(tempDir, dir)
		if err := Chmod(d, 0555); err != nil {
			t.Fatal(err)
		}

		// Defer changing the mode back so that the deferred
		// RemoveAll(tempDir) can succeed.
		defer Chmod(d, 0777)
	}

	err = RemoveAll(tempDir)
	if err == nil {
		t.Fatal("RemoveAll succeeded unexpectedly")
	}

	// The error should be of type *PathError.
	// see issue 30491 for details.
	if pathErr, ok := err.(*PathError); ok {
		want := filepath.Join(tempDir, "b", "y")
		if pathErr.Path != want {
			t.Errorf("RemoveAll(%q): err.Path=%q, want %q", tempDir, pathErr.Path, want)
		}
	} else {
		t.Errorf("RemoveAll(%q): error has type %T, want *fs.PathError", tempDir, err)
	}

	for _, dir := range dirs {
		_, err := Stat(filepath.Join(tempDir, dir))
		if inReadonly(dir) {
			if err != nil {
				t.Errorf("file %q was deleted but should still exist", dir)
			}
		} else {
			if err == nil {
				t.Errorf("file %q still exists but should have been deleted", dir)
			}
		}
	}
}

func TestRemoveUnreadableDir(t *testing.T) {
	switch runtime.GOOS {
	case "js":
		t.Skipf("skipping test on %s", runtime.GOOS)
	}

	if Getuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	t.Parallel()

	tempDir, err := os.MkdirTemp("", "TestRemoveAllButReadOnly-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tempDir)

	target := filepath.Join(tempDir, "d0", "d1", "d2")
	if err := MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Chmod(target, 0300); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAll(filepath.Join(tempDir, "d0")); err != nil {
		t.Fatal(err)
	}
}

// Issue 29921
func TestRemoveAllWithMoreErrorThanReqSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "TestRemoveAll-")
	if err != nil {
		t.Fatal(err)
	}
	defer RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "_TestRemoveAllWithMoreErrorThanReqSize_")

	// Make directory with 1025 read-only files.
	if err := MkdirAll(path, 0777); err != nil {
		t.Fatalf("MkdirAll %q: %s", path, err)
	}
	for i := 0; i < 1025; i++ {
		fpath := filepath.Join(path, fmt.Sprintf("file%d", i))
		fd, err := Create(fpath)
		if err != nil {
			t.Fatalf("create %q: %s", fpath, err)
		}
		fd.Close()
	}

	// Make the parent directory read-only. On some platforms, this is what
	// prevents os.Remove from removing the files within that directory.
	if err := Chmod(path, 0555); err != nil {
		t.Fatal(err)
	}
	defer Chmod(path, 0755)

	// This call should not hang, even on a platform that disallows file deletion
	// from read-only directories.
	err = RemoveAll(path)

	if Getuid() == 0 {
		// On many platforms, root can remove files from read-only directories.
		return
	}
	if err == nil {
		if runtime.GOOS == "windows" {
			// Marking a directory as read-only in Windows does not prevent the RemoveAll
			// from creating or removing files within it.
			return
		}
		t.Fatal("RemoveAll(<read-only directory>) = nil; want error")
	}

	dir, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()

	names, _ := dir.Readdirnames(1025)
	if len(names) < 1025 {
		t.Fatalf("RemoveAll(<read-only directory>) unexpectedly removed %d read-only files from that directory", 1025-len(names))
	}
}
