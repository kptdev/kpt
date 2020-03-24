package common

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func ReadFiles(source string, recursive bool) ([]string, error) {
	filePaths := make([]string, 0)
	if recursive {
		err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(info.Name()) == ".md" {
				filePaths = append(filePaths, path)
			}
			return nil
		})
		if err != nil {
			return filePaths, err
		}
	} else {
		files, err := ioutil.ReadDir(source)
		if err != nil {
			return filePaths, err
		}
		for _, info := range files {
			if filepath.Ext(info.Name()) == ".md" {
				path := filepath.Join(source, info.Name())
				filePaths = append(filePaths, path)
			}
		}
	}
	return filePaths, nil
}
