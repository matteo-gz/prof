package data

import (
	"errors"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/matteo-gz/prof/pkg/filex"
	"github.com/matteo-gz/prof/pkg/pproftype"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	mimeTrace = "application/octet-stream"
	mimePprof = "application/gzip"
	mimeTxt   = "text/plain"
)

func (f *file) getFileName(uri, ext string) (filename string, err error) {
	u2, err := url.Parse(uri)
	if err != nil {
		return
	}
	filename = strings.ReplaceAll(path.Base(u2.Path), "/", "_")
	filename += "." + ext
	return
}
func (f *file) getFileName2(name, ext string) (filename string, err error) {
	filename = name + "." + ext
	return
}

func (f *file) createFile(filename string, dir string, data []byte) (relativePath string, err error) {
	filePath := fmt.Sprintf("%s/%s", dir, filename)
	filePath = filepath.Join(filepath.Split(filePath))
	var (
		f1 *os.File
	)
	if filex.IsFileExist(filePath) {
		f.log.Errorf("modify file %s", filePath)
		f1, err = os.OpenFile(filePath, os.O_WRONLY, os.ModeAppend)
	} else {
		f.log.Debugf("create file %s", filePath)
		f1, err = os.Create(filePath)
	}
	if err != nil {
		return
	}
	if _, err = f1.Write(data); err != nil {
		return
	}
	return f.getRelDir(filePath), nil
}

func checkMimeByData(data []byte, contentType string) (string, error) {
	mtype := mimetype.Detect(data)
	if mimeTrace == mtype.String() && contentType != "" {
		return checkMime(contentType)
	}
	return checkMime(mtype.String())
}
func checkMime(string2 string) (string, error) {
	mtype2 := strings.Split(string2, ";")[0]
	t := map[string]string{
		mimeTrace: pproftype.ExtTrace,
		mimePprof: pproftype.ExtPprof,
		mimeTxt:   pproftype.ExtTxt,
	}
	for v, t2 := range t {
		if v == mtype2 {
			return t2, nil
		}
	}
	return "", errors.New("mime err")
}
func checkMimeByDir(dir string) (string, error) {
	mtype, err := mimetype.DetectFile(dir)
	if err != nil {
		return "", err
	}
	return checkMime(mtype.String())
}

func (f *file) dirName() string {
	t := time.Now()
	dateStr := t.Format("20060102")
	id := fmt.Sprintf("%d_%d_%d", t.Hour(), t.Minute(), t.Second())
	dir := f.getAbsDir(fmt.Sprintf("/%s/%s", dateStr, id))
	dir = filepath.Join(filepath.Split(dir))
	return dir
}
func (f *file) createDir() (string, error) {
	dir := f.dirName()
	if filex.IsDirExist(dir) {
		return dir, nil
	}
	f.log.Debugf("mkdir %s", dir)
	err := os.MkdirAll(dir, os.ModePerm)
	return dir, err
}
