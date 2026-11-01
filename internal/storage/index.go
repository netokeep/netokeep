package storage

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

var homeDir, _ = os.UserHomeDir()
var clientConfPath = filepath.Join(homeDir, ".config", "netokeep", "nk_settings.json")
var serverConfPath = filepath.Join(homeDir, ".config", "netokeep", "nk_server.json")

func LoadClientConf() (data *ClientProxy, err error) {
	contents, err := os.ReadFile(clientConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Client configuration file not found. Overwriting with default.")
			SaveClientConf(defaultClientConf)
			return &defaultClientConf.Proxy, nil
		}
		return nil, err
	}
	data = new(ClientProxy)
	if err := json.Unmarshal(contents, data); err != nil {
		log.Println("Failed to unmarshal client configuration. Overwriting with default.")
		if err := SaveClientConf(defaultClientConf); err != nil {
			log.Printf("Failed to save default client configuration: %v", err)
		}
		return &defaultClientConf.Proxy, nil
	}
	return data, nil
}

func LoadServerConf() (data *ServerRules, err error) {
	contents, err := os.ReadFile(serverConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Server configuration file not found. Overwriting with default.")
			SaveServerConf(defaultServerConf)
			return &defaultServerConf.Rules, nil
		}
		return nil, err
	}
	data = new(ServerRules)
	if err := json.Unmarshal(contents, data); err != nil {
		log.Println("Failed to unmarshal server configuration. Overwriting with default.")
		if err := SaveServerConf(defaultServerConf); err != nil {
			log.Printf("Failed to save default server configuration: %v", err)
		}
		return &defaultServerConf.Rules, nil
	}
	return data, nil
}

func SaveClientConf(conf *ClientConf) error {
	contents, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(clientConfPath, contents, 0644)
}

func SaveServerConf(conf *ServerConf) error {
	contents, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(serverConfPath, contents, 0644)
}

func ClearClientConf() error {
	return os.Remove(clientConfPath)
}

func ClearServerConf() error {
	return os.Remove(serverConfPath)
}
