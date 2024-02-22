package common

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
)

// const (
// 	bucketName = "groceries_images"
// )

func DeleteImageFromStorage(ctx context.Context, imageURL string, bucket_name string) error {
	objectName := getImageObjectNameFromURL(imageURL)
	fmt.Println(objectName)
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Cloud Storage client: %v", err)
	}
	defer client.Close()

	err = client.Bucket(bucket_name).Object(objectName).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete object from Cloud Storage: %v", err)
	}

	return nil
}
func getImageObjectNameFromURL(imageURL string) string {
	// Extract the object name from the image URL
	parts := strings.Split(imageURL, "/")
	return parts[len(parts)-1]
}
