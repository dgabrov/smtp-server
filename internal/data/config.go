package data

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

type DkimConfig struct {
	Enabled    bool
	PrivateKey string
	Selector   string
}

type QueueConfig struct {
	TimeBetweenLoads       int
	LoadSize               int
	MaxAttempts            int
	TimeBetweenAttempts    int
	SimultaneousProcessing int
}
type ConfigData struct {
	Domain             string
	ListenAddress      string
	ReadTimeoutSecond  int
	WriteTimeoutSecond int
	MaxMessageBytes    int64
	AllowInsecureAuth  bool
	Certificates       struct {
		Public  string
		Private string
	}

	Enabled587 bool
	Address587 string

	Db struct {
		Machine  string
		Port     int
		Login    string
		Password string
		Database string
	}
	Dkim DkimConfig
	Log  struct {
		Filename   string
		MaxSize    int
		MaxAge     int
		Compress   bool
		MaxBackups int
	}
	Queue QueueConfig
}

func LoadConfig() (ConfigData, error) {
	slog.Info("loading config data... ")
	configFileName := os.Getenv("CONFIG_FILE")
	bytes, err := os.ReadFile(configFileName)
	if err != nil {
		return ConfigData{}, err
	}

	var config ConfigData
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return ConfigData{}, err
	}

	slog.Info("config data successfully loaded")

	return config, nil
}

func LogConfig(config ConfigData) {
	slog.Info(fmt.Sprintf("domain: %s", config.Domain))
	slog.Info(fmt.Sprintf("listenAddress: %s", config.ListenAddress))
	slog.Info(fmt.Sprintf("readTimeoutSecond: %d", config.ReadTimeoutSecond))
	slog.Info(fmt.Sprintf("writeTimeoutSecond: %d", config.WriteTimeoutSecond))
	slog.Info(fmt.Sprintf("maxMessageBytes: %d", config.MaxMessageBytes))
	slog.Info(fmt.Sprintf("allowInsecureAuth: %v", config.AllowInsecureAuth))
	slog.Info(fmt.Sprintf("public key: %s", config.Certificates.Public))
	slog.Info(fmt.Sprintf("private key: %s", config.Certificates.Private))
	slog.Info(fmt.Sprintf("587 enabled: %t", config.Enabled587))
	slog.Info(fmt.Sprintf("587 listen address: %s", config.Address587))

	configDb := config.Db
	slog.Info(fmt.Sprintf("db machine: %s", configDb.Machine))
	slog.Info(fmt.Sprintf("db port: %d", configDb.Port))
	slog.Info(fmt.Sprintf("db login: %s", configDb.Login))
	slog.Info(fmt.Sprintf("db password length: %d", len(configDb.Password)))
	slog.Info(fmt.Sprintf("db database: %s", configDb.Database))

	configLog := config.Log
	slog.Info(fmt.Sprintf("log fileName: %s", configLog.Filename))
	slog.Info(fmt.Sprintf("log maxSize: %d", configLog.MaxSize))
	slog.Info(fmt.Sprintf("log maxAge: %d", configLog.MaxAge))
	slog.Info(fmt.Sprintf("log compress: %t", configLog.Compress))
	slog.Info(fmt.Sprintf("log maxBackups: %d", configLog.MaxBackups))

	configQueue := config.Queue
	slog.Info(fmt.Sprintf("log queue simultaneousProcessing: %d", configQueue.SimultaneousProcessing))
	slog.Info(fmt.Sprintf("log queue maxAttempts: %d", configQueue.MaxAttempts))
	slog.Info(fmt.Sprintf("log queue timeBetweenAttempts: %d", configQueue.TimeBetweenAttempts))
	slog.Info(fmt.Sprintf("log queue timeBetweenLoads: %d", configQueue.TimeBetweenLoads))
	slog.Info(fmt.Sprintf("log queue loadSize: %d", configQueue.LoadSize))

	dkimConfig := config.Dkim
	slog.Info(fmt.Sprintf("dkim enabled: %t", dkimConfig.Enabled))
	slog.Info(fmt.Sprintf("dkim private key: %s", dkimConfig.PrivateKey))
	slog.Info(fmt.Sprintf("dkim selector: %s", dkimConfig.Selector))
}
