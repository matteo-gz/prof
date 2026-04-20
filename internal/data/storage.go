package data

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/matteo-gz/prof/pkg/filex"
	"github.com/matteo-gz/prof/pkg/pproftype"
)

const (
	mimeTrace = "application/octet-stream"
	mimePprof = "application/gzip"
	mimeTxt   = "text/plain"
)

func (f *file) getFileName(uri string) (filename string, err error) {
	u2, err := url.Parse(uri)
	if err != nil {
		return
	}
	filename = strings.ReplaceAll(path.Base(u2.Path), "/", "_")
	q := u2.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := q.Get(k)
		if v == "" || v == "0" {
			continue
		}
		filename += "_" + k + "_" + v
	}
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
	defer f1.Close()
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
func checkMime(mimeStr string) (string, error) {
	mtype2 := strings.Split(mimeStr, ";")[0]
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
	id := fmt.Sprintf("%02d_%02d_%02d", t.Hour(), t.Minute(), t.Second())
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
