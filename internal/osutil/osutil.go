package osutil

import "os"

type FileSystem interface {
	Create(name string) (*os.File, error)
	MkdirAll(path string, perm os.FileMode) error
	Open(name string) (*os.File, error)
	Remove(name string) error
	Rename(oldpath, newpath string) error
	Stat(name string) (os.FileInfo, error)
}

type RealFS struct{}

func (RealFS) Create(name string) (*os.File, error)    { return os.Create(name) }
func (RealFS) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (RealFS) Open(name string) (*os.File, error)      { return os.Open(name) }
func (RealFS) Remove(name string) error                 { return os.Remove(name) }
func (RealFS) Rename(oldpath, newpath string) error     { return os.Rename(oldpath, newpath) }
func (RealFS) Stat(name string) (os.FileInfo, error)   { return os.Stat(name) }
