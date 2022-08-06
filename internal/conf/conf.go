package conf

import (
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

func CheckEnv(Env string) bool {
	if Env == EnvLocal ||
		Env == EnvProd {
		return true
	}
	return false
}
func Load(env string, flagPort, flagEnv, flagDir, flagPprofPort, flagLog string) (bs Bs, err error) {
	var in []byte
	if env == "" {
		bs = Bs{
			Server: &Server{
				Port:  flagPort,
				Port2: flagPprofPort,
			},
			Data: &Data{
				StorageDir: flagDir,
			},
			App: &App{
				Env: flagEnv,
				Log: flagLog,
			},
		}
	} else {
		if in, err = ioutil.ReadFile(env); err == nil {
			err = yaml.Unmarshal(in, &bs)
		}
	}

	if !CheckEnv(bs.App.Env) {
		err = errors.New("config env err")
		return
	}
	if bs.Data.StorageDir == "" {
		err = errors.New("config StorageDir empty")
		return
	}
	if bs.App.Log == "" {
		err = errors.New("config log empty")
		return
	}
	return
}
