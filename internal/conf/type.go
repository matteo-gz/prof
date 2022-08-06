package conf

// todo support protoc

type Data struct {
	StorageDir string `yaml:"StorageDir"`
}
type Server struct {
	Port  string `yaml:"Port"`
	Port2 string `yaml:"Port2"`
}
type App struct {
	Env string `yaml:"Env"`
	Log string `yaml:"Log"`
}
type Bs struct {
	Server *Server `yaml:"Server"`
	Data   *Data   `yaml:"Data"`
	App    *App    `yaml:"App"`
}
