package utils

import (
	"context"

	"cloud.google.com/go/storage"
)

func CreateStorageClient() (*storage.Client, error) {
	ctx := context.Background()

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return storageClient, nil
}
