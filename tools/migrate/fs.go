package migrate

import (
	"bufio"
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
)

type FS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

const IgnoreMarker = "-- migration:ignore"

var _ FS = (*SourceWrapper)(nil)

type SourceWrapper struct {
	fsys fs.FS
}

func NewSourceWrapper(fsys fs.FS) *SourceWrapper {
	return &SourceWrapper{fsys: fsys}
}

func (s *SourceWrapper) Open(name string) (fs.File, error) {
	return s.fsys.Open(name)
}

func (s *SourceWrapper) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(s.fsys, path)
	if err != nil {
		return nil, err
	}

	results := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		filePath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			r, err := s.ReadDir(filePath)
			if err != nil {
				return nil, err
			}

			results = append(results, r...)
		}

		var b []byte

		b, err = fs.ReadFile(s.fsys, filePath)
		if err != nil {
			return nil, err
		}

		var ignore bool

		scanner := bufio.NewScanner(bytes.NewBuffer(b))
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), IgnoreMarker) {
				ignore = true
				break
			}
		}

		if ignore {
			continue
		}

		results = append(results, entry)
	}

	return results, nil
}

func (s *SourceWrapper) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(s.fsys, name)
}
