package biz

import (
	"io"
	"mime/multipart"
)

func (uc *Usecase) DealUpload(file multipart.File, filename string) (s1 string, err error) {
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return
	}
	return uc.repo.CreateFileByUpload(filename, data)
}
