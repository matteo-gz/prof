package filex

import (
	"os"
	"path/filepath"
)

func Path(dir string) (abs string, err error) {
	if !IsDirExist(dir) {
		err = os.MkdirAll(dir, os.ModePerm)
	}
	if err != nil {
		return
	}
	return filepath.Abs(dir)
}
func IsFileExist(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fileInfo.IsDir() {
		return false
	}
	return true
}
func IsDirExist(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fileInfo.IsDir() {
		return true
	}
	return false
}
