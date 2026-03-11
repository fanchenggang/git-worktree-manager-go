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

func LoadHistory(configDir string) ([]MergeHistory, error) {
	historyPath := filepath.Join(configDir, "history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []MergeHistory{}, nil
		}
		return nil, err
	}

	var history []MergeHistory
	err = json.Unmarshal(data, &history)
	if err != nil {
		return nil, err
	}

	return history, nil
}

func SaveHistory(configDir string, history []MergeHistory) error {
	historyPath := filepath.Join(configDir, "history.json")

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(historyPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func AddHistory(configDir string, entry MergeHistory) error {
	history, err := LoadHistory(configDir)
	if err != nil {
		history = []MergeHistory{}
	}

	history = append([]MergeHistory{entry}, history...)

	// Keep only last 50 entries
	if len(history) > 50 {
		history = history[:50]
	}

	return SaveHistory(configDir, history)
}
