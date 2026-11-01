package storage

type ClientProxy struct {
	Type      string   `json:"type"`
	Addr      string   `json:"addr"`
	Port      uint16   `json:"port"`
	AllowList []string `json:"allow"`
}
type ClientConf struct {
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Proxy       ClientProxy `json:"proxy"`
}

var defaultClientConf = &ClientConf{
	Version:     "1.0.0",
	Description: "Default client configuration",
	Proxy: ClientProxy{
		Type: "socks5",
		Addr: "localhost",
		Port: 1080,
		AllowList: []string{
			"*.openai.com",
			"*.anthropic.com",
			"*.claude.ai",
			"*.claude.com",
			"*.googleapis.com",
		},
	},
}
