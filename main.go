package main

import (
	"log/slog"

	"github.com/dgb9/smtp-server/internal/start"
)

func main() {
	err := start.Start()

	if err != nil {
		slog.Info(err.Error())
	}
}
