package data

import (
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/pkg/filex"
)

type repo struct {
	data *Data
	log  *log.Helper
}

func NewRepo(data *Data, logger log.Logger) biz.Repo {
	return &repo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (rp *repo) GetPortByDir(string2 string) (port int, err error) {
	dir := rp.data.file.getAbsDir(string2)
	if a := filex.IsFileExist(dir); !a {
		return 0, errors.New("file not exist")
	}
	return rp.data.task.getPortByDir(dir)
}
func (rp *repo) GetAbsDir(string2 string) string {
	return rp.data.file.getAbsDir(string2)
}

func (rp *repo) GetFileList(date string) (list, files []string, err error) {
	return rp.data.file.getFileList(date)
}
func (rp *repo) GetFileType(dir string) string {
	return getFileType(dir)
}

func (rp *repo) CreateFile(uri string, data []byte) (RelativePath string, err error) {
	var ext string
	if ext, err = checkMimeByData(data); err != nil {
		return
	}
	filename, err := rp.data.file.getFileName(uri, ext)
	if err != nil {
		return
	}
	dir, err := rp.data.file.createDir()
	if err != nil {
		return
	}
	return rp.data.file.createFile(filename, dir, data)
}
func (rp *repo) CreateFileByUpload(fileName string, data []byte) (relativePath string, err error) {
	var ext string
	if ext, err = checkMimeByData(data); err != nil {
		return
	}
	filename, err := rp.data.file.getFileName2(fileName, ext)
	if err != nil {
		return
	}
	dir, err := rp.data.file.createDir()
	if err != nil {
		return
	}
	return rp.data.file.createFile(filename, dir, data)
}
