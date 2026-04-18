package data

import (
	"fmt"
	"os"
	"path"
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
	ext := path.Ext(dir)
	if ext != "" {
		ext = ext[1:]
	}
	if pproftype.ExtTrace == ext {
		return pproftype.ExtTrace
	} else if pproftype.ExtTxt == ext {
		return pproftype.ExtTxt
	} else if pproftype.ExtPprof == ext {
		return pproftype.ExtPprof
	} else {
		return pproftype.ExtUnknown
	}
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
			files = append(files, biz.FileInfo{Name: fi.Name(), Size: fi.Size()})
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(list)))
	return list, files, nil
}
