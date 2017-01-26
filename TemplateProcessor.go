package goboots

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type TemplateProcessor interface {
	Walk(root string, walkFn filepath.WalkFunc) error
	ReadFile(filename string) ([]byte, error)
}

type defaultTemplateProcessor struct {
}

func (d *defaultTemplateProcessor) Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

func (d *defaultTemplateProcessor) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m *mockFileInfo) Name() string {
	return m.name
}

func (m *mockFileInfo) Size() int64 {
	return m.size
}

func (m *mockFileInfo) Mode() os.FileMode {
	return m.mode
}

func (m *mockFileInfo) ModTime() time.Time {
	return m.modTime
}

func (m *mockFileInfo) IsDir() bool {
	return m.isDir
}

func (m *mockFileInfo) Sys() interface{} {
	return nil
}

func NewMockFileInfo(name string, size int64, mode os.FileMode, modTime time.Time, isDir bool) os.FileInfo {
	m := &mockFileInfo{}
	m.name = name
	m.size = size
	m.mode = mode
	m.modTime = modTime
	m.isDir = isDir
	return m
}
