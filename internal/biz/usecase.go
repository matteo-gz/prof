package biz

import (
	"encoding/base64"
	"github.com/go-kratos/kratos/v2/log"
)

type Usecase struct {
	repo            Repo
	log             *log.Helper
	denyPrivateIP   bool
	samplingSeconds int
	deltaSeconds    int
	traceSeconds    int
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

func (uc *Usecase) GetFileList(date string) (list []string, files []FileInfo, err error) {
	return uc.repo.GetFileList(date)
}
func (uc *Usecase) GetBatchGroups(date string) ([]BatchGroup, error) {
	return uc.repo.GetBatchGroups(date)
}
func (uc *Usecase) GetFileType(dir string) string {
	return uc.repo.GetFileType(dir)
}

func (uc *Usecase) AnalyzeTop(enBase64Dir string, n int, sampleIdx int, cumSort bool) (*TopResult, error) {
	dir, err := base64.StdEncoding.DecodeString(enBase64Dir)
	if err != nil {
		return nil, err
	}
	absPath := uc.repo.GetAbsDir(string(dir))
	result, err := AnalyzeTop(absPath, n, sampleIdx, cumSort)
	if err != nil {
		return nil, err
	}
	result.Path = string(dir)
	return result, nil
}

func (uc *Usecase) AnalyzeSource(enBase64Dir, filter string, sampleIdx, margin int, sourcePath string, maxFiles int) (*SourceResult, error) {
	dir, err := base64.StdEncoding.DecodeString(enBase64Dir)
	if err != nil {
		return nil, err
	}
	absPath := uc.repo.GetAbsDir(string(dir))
	result, err := AnalyzeSource(absPath, filter, sampleIdx, margin, sourcePath, maxFiles)
	if err != nil {
		return nil, err
	}
	result.Path = string(dir)
	return result, nil
}

func (uc *Usecase) AnalyzePeek(enBase64Dir, filter string, sampleIdx int) (*PeekResult, error) {
	dir, err := base64.StdEncoding.DecodeString(enBase64Dir)
	if err != nil {
		return nil, err
	}
	absPath := uc.repo.GetAbsDir(string(dir))
	result, err := AnalyzePeek(absPath, filter, sampleIdx)
	if err != nil {
		return nil, err
	}
	result.Path = string(dir)
	return result, nil
}

func (uc *Usecase) AnalyzeFlame(enBase64Dir string, sampleIdx int, trimPath, sourcePath string) (*FlameResult, error) {
	dir, err := base64.StdEncoding.DecodeString(enBase64Dir)
	if err != nil {
		return nil, err
	}
	absPath := uc.repo.GetAbsDir(string(dir))
	result, err := AnalyzeFlame(absPath, sampleIdx, trimPath, sourcePath)
	if err != nil {
		return nil, err
	}
	result.Path = string(dir)
	return result, nil
}

func (uc *Usecase) AnalyzeFlameLayout(enBase64Dir string, sampleIdx int, trimPath, sourcePath string) (*FlameLayoutResult, error) {
	dir, err := base64.StdEncoding.DecodeString(enBase64Dir)
	if err != nil {
		return nil, err
	}
	absPath := uc.repo.GetAbsDir(string(dir))
	result, err := AnalyzeFlameLayout(absPath, sampleIdx, trimPath, sourcePath)
	if err != nil {
		return nil, err
	}
	result.Path = string(dir)
	return result, nil
}
