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

func (rp *repo) GetPortByDir(relPath string) (port int, err error) {
	dir := rp.data.file.getAbsDir(relPath)
	if a := filex.IsFileExist(dir); !a {
		return 0, errors.New("file not exist")
	}
	return rp.data.task.getPortByDir(dir)
}

func (rp *repo) GetBatchGroups(date string) (groups []biz.BatchGroup, err error) {
	dirs, _, err := rp.GetFileList(date)
	if err != nil {
		return
	}
	for _, d := range dirs {
		subDir := date + "/" + d
		_, subFiles, err2 := rp.GetFileList(subDir)
		if err2 != nil || len(subFiles) == 0 {
			continue
		}
		groups = append(groups, biz.BatchGroup{
			Label: d,
			Dir:   subDir,
			Files: subFiles,
		})
	}
	return
}

func (rp *repo) GetPortByDiff(baseFile, compareFile string) (port int, err error) {
	base := rp.data.file.getAbsDir(baseFile)
	compare := rp.data.file.getAbsDir(compareFile)
	if !filex.IsFileExist(base) {
		return 0, errors.New("base file not exist")
	}
	if !filex.IsFileExist(compare) {
		return 0, errors.New("compare file not exist")
	}
	return rp.data.task.getPortByDiff(base, compare)
}
func (rp *repo) GetAbsDir(relPath string) string {
	return rp.data.file.getAbsDir(relPath)
}

func (rp *repo) GetFileList(date string) (list []string, files []biz.FileInfo, err error) {
	return rp.data.file.getFileList(date)
}
func (rp *repo) GetFileType(dir string) string {
	return getFileType(rp.data.file.getAbsDir(dir))
}

func (rp *repo) PreAllocDir() (string, error) {
	return rp.data.file.createDir()
}

func (rp *repo) CreateFileInDir(absDir, uri, contentType string, data []byte) (RelativePath string, err error) {
	if _, err = checkMimeByData(data, contentType); err != nil {
		return
	}
	filename, err := rp.data.file.getFileName(uri)
	if err != nil {
		return
	}
	return rp.data.file.createFile(filename, absDir, data)
}

func (rp *repo) CreateFile(uri, contentType string, data []byte) (RelativePath string, err error) {
	if _, err = checkMimeByData(data, contentType); err != nil {
		return
	}
	filename, err := rp.data.file.getFileName(uri)
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
	if _, err = checkMimeByData(data, ""); err != nil {
		return
	}
	dir, err := rp.data.file.createDir()
	if err != nil {
		return
	}
	return rp.data.file.createFile(fileName, dir, data)
}
