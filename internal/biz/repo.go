package biz

import "github.com/go-kratos/kratos/v2/log"

type Repo interface {
	GetPortByDir(relPath string) (port int, err error)
	GetAbsDir(relPath string) string
	GetFileList(date string) (list, files []string, err error)
	GetFileType(dir string) string
	CreateFile(url string, contentType string, data []byte) (relativePath string, err error)
	CreateFileByUpload(fileName string, data []byte) (relativePath string, err error)
}

func NewUsecase(repo Repo, logger log.Logger, denyPrivateIP bool, samplingSeconds, deltaSeconds, traceSeconds int) *Usecase {
	if samplingSeconds <= 0 {
		samplingSeconds = 30
	}
	if deltaSeconds <= 0 {
		deltaSeconds = 10
	}
	if traceSeconds <= 0 {
		traceSeconds = 5
	}
	return &Usecase{
		repo:            repo,
		log:             log.NewHelper(logger),
		denyPrivateIP:   denyPrivateIP,
		samplingSeconds: samplingSeconds,
		deltaSeconds:    deltaSeconds,
		traceSeconds:    traceSeconds,
	}
}
