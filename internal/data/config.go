package data

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

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
	Db struct {
		Machine  string
		Port     int
		Login    string
		Password string
		Database string
	}
	Log struct {
		Filename   string
		MaxSize    int
		MaxAge     int
		Compress   bool
		MaxBackups int
	}
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
	slog.Info(fmt.Sprintf("db machine: %s", config.Db.Machine))
	slog.Info(fmt.Sprintf("db port: %d", config.Db.Port))
	slog.Info(fmt.Sprintf("db login: %s", config.Db.Login))
	slog.Info(fmt.Sprintf("db password length: %d", len(config.Db.Password)))
	slog.Info(fmt.Sprintf("db database: %s", config.Db.Database))

	slog.Info(fmt.Sprintf("log fileName: %s", config.Log.Filename))
	slog.Info(fmt.Sprintf("log maxSize: %d", config.Log.MaxSize))
	slog.Info(fmt.Sprintf("log maxAge: %d", config.Log.MaxAge))
	slog.Info(fmt.Sprintf("log compress: %t", config.Log.Compress))
	slog.Info(fmt.Sprintf("log maxBackups: %d", config.Log.MaxBackups))
}
