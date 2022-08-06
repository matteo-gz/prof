package biz

import (
	"encoding/base64"
	"github.com/go-kratos/kratos/v2/log"
)

type Usecase struct {
	repo Repo
	log  *log.Helper
}

func (uc *Usecase) getPortByDir(enBase64Dir string) (usePort int, err error) {
	dir, err := base64.StdEncoding.DecodeString(enBase64Dir)
	if err != nil {
		return
	}
	return uc.repo.GetPortByDir(string(dir))
}
func (uc *Usecase) GetAbsDir(s string) string {
	return uc.repo.GetAbsDir(s)
}

func (uc *Usecase) GetFileList(date string) (list, files []string, err error) {
	return uc.repo.GetFileList(date)
}
func (uc *Usecase) GetFileType(dir string) string {
	return uc.repo.GetFileType(dir)
}
