// Copyright ©2015 The Go Authors
// Copyright ©2015 Steve Francia <spf@spf13.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package i18n

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// helper interfaces to enable compability with virtual filesystems
// compatible with Afero:
// https://github.com/spf13/afero

// File represents a file in the filesystem.
type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Writer
	io.WriterAt

	Name() string
	Readdir(count int) ([]os.FileInfo, error)
	Readdirnames(n int) ([]string, error)
	Stat() (os.FileInfo, error)
	Sync() error
	Truncate(size int64) error
	WriteString(s string) (ret int, err error)
}

// Fs is the filesystem interface.
//
// Any simulated or real filesystem should implement this interface.
type Fs interface {
	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	Create(name string) (File, error)

	// Mkdir creates a directory in the filesystem, return an error if any
	// happens.
	Mkdir(name string, perm os.FileMode) error

	// MkdirAll creates a directory path and all parents that does not exist
	// yet.
	MkdirAll(path string, perm os.FileMode) error

	// Open opens a file, returning it or an error, if any happens.
	Open(name string) (File, error)

	// OpenFile opens a file using the given flags and the given mode.
	OpenFile(name string, flag int, perm os.FileMode) (File, error)

	// Remove removes a file identified by name, returning an error, if any
	// happens.
	Remove(name string) error

	// RemoveAll removes a directory path and any children it contains. It
	// does not fail if the path does not exist (return nil).
	RemoveAll(path string) error

	// Rename renames a file.
	Rename(oldname, newname string) error

	// Stat returns a FileInfo describing the named file, or an error, if any
	// happens.
	Stat(name string) (os.FileInfo, error)

	// The name of this FileSystem
	Name() string

	//Chmod changes the mode of the named file to mode.
	Chmod(name string, mode os.FileMode) error

	//Chtimes changes the access and modification times of the named file
	Chtimes(name string, atime time.Time, mtime time.Time) error
}

type osfs int8

func (_ *osfs) Create(name string) (File, error) {
	return os.Create(name)
}

func (_ *osfs) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func (_ *osfs) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (_ *osfs) Open(name string) (File, error) {
	return os.Open(name)
}

func (_ *osfs) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

func (_ *osfs) Remove(name string) error {
	return os.Remove(name)
}

func (_ *osfs) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (_ *osfs) Rename(oldname, newname string) error {
	return os.Rename(oldname, newname)
}

func (_ *osfs) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (_ *osfs) Name() string {
	return "system"
}

func (_ *osfs) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (_ *osfs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
}

func OsFS() Fs {
	v := osfs(1)
	return &v
}

func Walk(fs Fs, root string, walkFn filepath.WalkFunc) error {
	info, err := stat(fs, root)
	if err != nil {
		return walkFn(root, nil, err)
	}
	return walk(fs, root, info, walkFn)
}

// walk recursively descends path, calling walkFn
// adapted from https://golang.org/src/path/filepath/path.go
func walk(fs Fs, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(path, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := readDirNames(fs, path)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := stat(fs, filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(fs, filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
// adapted from https://golang.org/src/path/filepath/path.go
func readDirNames(fs Fs, dirname string) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func stat(fs Fs, path string) (info os.FileInfo, err error) {
	_, ok := fs.(*osfs)
	if ok {
		info, err = os.Lstat(path)
	} else {
		info, err = fs.Stat(path)
	}
	return
}
