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
	Env             string `yaml:"Env"`
	Log             string `yaml:"Log"`
	DenyPrivateIP   bool   `yaml:"DenyPrivateIP"`
	SamplingSeconds int    `yaml:"SamplingSeconds"`
	DeltaSeconds    int    `yaml:"DeltaSeconds"`
	TraceSeconds    int    `yaml:"TraceSeconds"`
}
type PluginsConfig struct {
	I18nZh       bool `yaml:"i18n_zh"`
	SourceFold   bool `yaml:"source_fold"`
	PeekFold     bool `yaml:"peek_fold"`
	GraphExplain bool `yaml:"graph_explain"`
}

// Mcp configures the stdio MCP subprocess (`prof … mcp`).
type Mcp struct {
	// APIBase is the full Prof HTTP root used by MCP tools (no trailing slash).
	// If empty, defaults to http://127.0.0.1:{Server.Port}.
	APIBase string `yaml:"APIBase"`
}

type Bs struct {
	Server  *Server        `yaml:"Server"`
	Data    *Data          `yaml:"Data"`
	App     *App           `yaml:"App"`
	Plugins *PluginsConfig `yaml:"Plugins"`
	Mcp     *Mcp           `yaml:"Mcp"`
}
