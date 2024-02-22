package utils

import (
	"context"

	"cloud.google.com/go/firestore"
)

func CreateFirestoreClient() (*firestore.Client, error) {
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, "capstore-takeoff")
	if err != nil {
		return nil, err
	}

	return client, nil
}
