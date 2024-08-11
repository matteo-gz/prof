package biz

import "github.com/go-kratos/kratos/v2/log"

type Repo interface {
	GetPortByDir(string2 string) (port int, err error)
	GetAbsDir(string2 string) string
	GetFileList(date string) (list, files []string, err error)
	GetFileType(dir string) string
	CreateFile(url string, contentType string, data []byte) (relativePath string, err error)
	CreateFileByUpload(fileName string, data []byte) (relativePath string, err error)
}

func NewUsecase(repo Repo, logger log.Logger) *Usecase {
	return &Usecase{repo: repo, log: log.NewHelper(logger)}
}
