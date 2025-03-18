package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Path     string            `yaml:"path"`
	Database Database          `yaml:"database"`
	Schemas  map[string]Schema `yaml:"schemas"`
}

type Database struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SslMode  string `yaml:"ssl_mode"`
	Params   string `yaml:"params"`
}

type Schema struct {
	Tables Tables `yaml:"tables"`
}

type Tables struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

func Parse(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.Database.Host == "" {
		config.Database.Host = os.Getenv("DATABASE_HOST")
	}

	if config.Database.Password == "" {
		config.Database.Password = os.Getenv("DATABASE_PASS")
	}

	return &config, nil
}
