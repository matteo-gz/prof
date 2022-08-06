package main

import (
	"flag"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/internal/conf"
	"github.com/matteo-gz/prof/internal/server"
	"github.com/matteo-gz/prof/pkg/appx"
	"github.com/matteo-gz/prof/pkg/logx"
)

var (
	flagConf      = ""
	flagPort      = ""
	flagPprofPort = ""
	flagEnv       = ""
	flagDir       = ""
	flagLog       = ""
)

func init() {
	flag.StringVar(&flagConf, "c", "", `config yaml file`)
	flag.StringVar(&flagPort, "port", "8201", `web server port`)
	flag.StringVar(&flagPprofPort, "port2", "8202", `pprof server port`)
	flag.StringVar(&flagEnv, "env", "prod", `env value, option:local,prod`)
	flag.StringVar(&flagDir, "dir", "./storage", `profile storage dir`)
	flag.StringVar(&flagLog, "log", "./logs", `log  dir`)

}
func main() {
	flag.Parse()
	bs, err := conf.Load(flagConf, flagPort, flagEnv, flagDir, flagPprofPort, flagLog)
	if err != nil {
		panic(err)
	}
	l := logx.Logger(bs.App.Env, bs.App.Log)
	defer func() {
		if err2 := l.Sync(); err2 != nil {
			fmt.Println(err2)
		}
	}()
	app, cleanup, err := wireApp(&bs, bs.Server, bs.Data, l)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	if err2 := app.Run(); err2 != nil {
		panic(err2)
	}
}
func newApp(logger log.Logger, hsx server.InterFace) *appx.App {
	return appx.New(logger, hsx)
}
