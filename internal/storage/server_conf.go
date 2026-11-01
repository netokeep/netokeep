package storage

type ServerRules struct {
	Default   string   `json:"default"`
	AllowList []string `json:"allow"`
	DenyList  []string `json:"deny"`
}

type ServerConf struct {
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Rules       ServerRules `json:"rules"`
}

var defaultServerConf = &ServerConf{
	Version:     "1.0.0",
	Description: "Forward rules for the netokeep server.",
	Rules: ServerRules{
		Default: "deny",
		AllowList: []string{
			"*.github.com",
			"*.githubusercontent.com",
			"*.pytorch.org",
			"*.huggingface.co",
			"*.modelscope.cn",
			"*.deepseek.com",
			"*.moonshot.cn",
			"*.openai.com",
			"*.anthropic.com",
			"*.claude.ai",
			"*.claude.com",
			"*.googleapis.com",
			"*.google.com",
			"*.edu.cn",
			"arxiv.org",
			"pypi.org",
			"files.pythonhosted.org",
			"*.nvidia.com",
			"*.nvidia.cn",
		},
		DenyList: []string{
			"localhost",
			"127.0.0.1",
			"::1",
			"fc00::/7",
			"fe80::/10",
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"169.254.0.0/16",
			"*.local",
		},
	},
}
