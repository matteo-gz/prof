package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/matteo-gz/prof/internal/conf"
)

var ProviderSet = wire.NewSet(NewData, NewRepo)

type Data struct {
	file *file
	task *task
	log  *log.Helper
}

func NewData(c *conf.Data, logger log.Logger) (dataData *Data, cleanup func(), err error) {
	dataData = &Data{
		log: log.NewHelper(logger),
	}
	dataData.file, err = newFile(c.StorageDir, logger)
	if err != nil {
		return
	}
	dataData.task = newTask(logger)
	cleanup = func() {
		log.NewHelper(logger).Info("closing the data resources")
		dataData.task.close()
	}
	return
}
func newTask(logger log.Logger) *task {
	t := &task{
		list:       make(map[string]*Proxy),
		timerCheck: 60,
		log:        log.NewHelper(logger),
	}
	go t.Timer()
	return t
}
func newProxy(dir string, l *log.Helper) *Proxy {
	return &Proxy{
		dir:      dir,
		fileType: getFileType(dir),
		lifeTime: 60 * 30, // 30 min
		log:      l,
	}
}
