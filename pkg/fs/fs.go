package fs

import (
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

// ScanDirectoriesForFiles walks the file trees for each entry in directories,
// and returns a list of all files with the ext for an extension, or an error
// if one is encountered
func ScanDirectoriesForFiles(ext string, directories []string) ([]string, error) {
	var files []string
	extLen := len(ext)
	for _, dir := range directories {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && path[len(path)-extLen-1:] == "."+ext {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "error scanning %s for files", dir)
		}
	}

	return files, nil
}
