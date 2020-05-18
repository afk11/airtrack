package fs

import (
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

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
