package utils

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/logging"
)

var (
	Logger *logging.Logger
)

func InitLogger() {
	ctx := context.Background()
	client, err := logging.NewClient(ctx, "capstore-takeoff")
	if err != nil {
		log.Fatalf("Failed to create logging client: %v", err)
	}
	defer client.Close()

	Logger = client.Logger("project-log")

	log.SetOutput(os.Stdout)
}

func InfoLog(message string) {
	Logger.Log(logging.Entry{Payload: message, Severity: logging.Info})
}

func ErrorLog(err error) {
	Logger.Log(logging.Entry{Payload: err.Error(), Severity: logging.Error})
}
