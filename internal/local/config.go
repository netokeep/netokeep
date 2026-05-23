package local

import (
	"encoding/json"
	"os"
	"path/filepath"

	_ "embed"
)

//go:embed nks_settings.json
var nksConfig []byte

//go:embed nk_settings.json
var nkConfig []byte

type RuleConfig struct {
	Default   string   `json:"default"`
	AllowList []string `json:"allow"`
	DenyList  []string `json:"deny"`
}

type NksConfig struct {
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Rules       RuleConfig `json:"rules"`
}

type ProxyConfig struct {
	Type      string   `json:"type"`
	Addr      string   `json:"address"`
	Port      int      `json:"port"`
	AllowList []string `json:"allow"`
}

type NkConfig struct {
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Proxy       ProxyConfig `json:"proxy"`
}

func nksConfigPath() string { return filepath.Join(configDir(), "nks_settings.json") }
func nkConfigPath() string  { return filepath.Join(configDir(), "nk_settings.json") }

func initNksConfig() error {
	return os.WriteFile(nksConfigPath(), nksConfig, 0644)
}

func initNkConfig() error {
	return os.WriteFile(nkConfigPath(), nkConfig, 0644)
}

func LoadNksConfig() (*NksConfig, error) {
	nksPath := filepath.Join(configDir(), "nks_settings.json")
	nksData, err := os.ReadFile(nksPath)
	if os.IsNotExist(err) {
		if err := initNksConfig(); err != nil {
			return nil, err
		}
		return LoadNksConfig()
	}

	if err != nil {
		return nil, err
	}
	var nksS NksConfig
	if err := json.Unmarshal(nksData, &nksS); err != nil {
		return nil, err
	}
	return &nksS, nil
}

func LoadNkConfig() (*NkConfig, error) {
	nkPath := filepath.Join(configDir(), "nk_settings.json")
	nkData, err := os.ReadFile(nkPath)
	if os.IsNotExist(err) {
		if err := initNkConfig(); err != nil {
			return nil, err
		}
		return LoadNkConfig()
	}

	if err != nil {
		return nil, err
	}
	var nkS NkConfig
	if err := json.Unmarshal(nkData, &nkS); err != nil {
		return nil, err
	}
	return &nkS, nil
}

func RemoveConfigs() error {
	if err := os.Remove(nkConfigPath()); err != nil {
		return err
	}
	if err := os.Remove(nksConfigPath()); err != nil {
		return err
	}
	return nil
}
