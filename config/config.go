package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
)

// Config represents the configuration for PostgreSQL connection
type PostgreConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	UseSSL   bool   `json:"useSSL"`
}

// LoadPostgreConfig reads the configuration from a JSON file and returns a PostgreConfig object
func LoadPostgreConfig(filePath string) (*PostgreConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Errorln("無法打開配置檔案")
		return nil, err
	}
	defer file.Close()

	config := &PostgreConfig{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		log.WithError(err).Errorln("解析配置檔案失敗")
		return nil, err
	}

	return config, nil
}
