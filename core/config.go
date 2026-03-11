package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Projects []string `json:"projects"`
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".gitwt")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return "", err
	}

	return configDir, nil
}

func LoadProjects(configDir string) ([]string, error) {
	configPath := filepath.Join(configDir, "projects.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config.Projects, nil
}

func saveProjects(configDir string, projects []string) error {
	configPath := filepath.Join(configDir, "projects.json")

	config := Config{
		Projects: projects,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
