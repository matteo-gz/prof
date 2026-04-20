package data

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/pkg/filex"
	"github.com/matteo-gz/prof/pkg/pproftype"
)

type file struct {
	dir string
	log *log.Helper
}

func newFile(dir string, logger log.Logger) (*file, error) {
	dir, err := filex.Path(dir)
	if err != nil {
		fmt.Println("file dir err", err.Error())
		return nil, err
	}
	f := &file{
		dir: dir,
		log: log.NewHelper(logger),
	}
	f.log.Infof("file dir: %s", dir)
	return f, nil
}
func (f *file) getRelDir(absPath string) string {
	return strings.ReplaceAll(absPath, f.dir, "")
}
func (f *file) getAbsDir(relPath string) string {
	return f.dir + relPath
}
func getFileType(dir string) (fileType string) {
	sniffed, err := checkMimeByDir(dir)
	if err != nil {
		return pproftype.ExtUnknown
	}
	return sniffed
}
func (f *file) getFileList(date string) (list []string, files []biz.FileInfo, err error) {
	final := f.getAbsDir(date)
	f1, err := os.OpenFile(final, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return
	}
	defer f1.Close()
	info, err := f1.Readdir(-1)
	if err != nil {
		return
	}
	for _, fi := range info {
		if fi.IsDir() {
			list = append(list, fi.Name())
		} else {
			absFile := final + string(os.PathSeparator) + fi.Name()
			files = append(files, biz.FileInfo{
				Name: fi.Name(),
				Size: fi.Size(),
				Type: getFileType(absFile),
			})
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(list)))
	return list, files, nil
}
