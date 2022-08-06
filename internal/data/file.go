package data

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/pkg/filex"
	"github.com/matteo-gz/prof/pkg/pproftype"
	"os"
	"path"
	"strings"
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
func (f *file) getRelDir(string2 string) string {
	return strings.ReplaceAll(string2, f.dir, "")
}
func (f *file) getAbsDir(string2 string) string {
	return f.dir + string2
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
func (f *file) getFileList(date string) (list, files []string, err error) {
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
	for _, fileInfo := range info {
		if fileInfo.IsDir() {
			list = append(list, fileInfo.Name())
		} else {
			files = append(files, fileInfo.Name())
		}
	}
	return list, files, nil
}
