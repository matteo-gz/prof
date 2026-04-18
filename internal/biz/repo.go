package biz

import "github.com/go-kratos/kratos/v2/log"

// FileInfo holds name and size for a file in the storage directory.
type FileInfo struct {
	Name string
	Size int64
}

// BatchGroup holds files from one collection batch under a date directory.
type BatchGroup struct {
	Label string     // e.g. "23_39_42"
	Dir   string     // relative path, e.g. "/20260415/23_39_42"
	Files []FileInfo
}

type Repo interface {
	GetPortByDir(relPath string) (port int, err error)
	GetPortByDiff(baseFile, compareFile string) (port int, err error)
	GetAbsDir(relPath string) string
	GetFileList(date string) (list []string, files []FileInfo, err error)
	GetBatchGroups(date string) ([]BatchGroup, error)
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
