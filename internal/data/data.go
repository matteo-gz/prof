package data

import (
	"context"

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
const (
	timerInterval = 60      // seconds between timer checks
	proxyLifetime = 60 * 30 // 30 minutes
)

func newTask(logger log.Logger) *task {
	ctx, cancel := context.WithCancel(context.Background())
	t := &task{
		list:       make(map[string]*Proxy),
		timerCheck: timerInterval,
		log:        log.NewHelper(logger),
		ctx:        ctx,
		cancel:     cancel,
	}
	go t.Timer()
	return t
}
func newProxy(dir string, baseFile string, l *log.Helper) *Proxy {
	return &Proxy{
		dir:      dir,
		baseFile: baseFile,
		fileType: getFileType(dir),
		lifeTime: proxyLifetime,
		log:      l,
		CmdOut:   newRingBuffer(maxCmdBufSize),
		CmdErr:   newRingBuffer(maxCmdBufSize),
	}
}
