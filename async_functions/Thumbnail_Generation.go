package async_functions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"

	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"github.com/disintegration/imaging"
)

const (
	bucketName = "thumbnail_images_bucket"
)

type ThumbnailFileContent struct {
	FileURL string `json:"fileURL"`
	DocId   int    `json:"ID"`
}

var (
	loggers       *logging.Logger
	storageClient *storage.Client
)

func init() {
	ctx := context.Background()

	storageClient, _ = storage.NewClient(ctx)

}
func GenerateThumbnail(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()
	loggingClient, _ := logging.NewClient(ctx, "capstore-takeoff")
	// if err != nil {
	// 	log.Fatalf("Failed to create logging client: %v", err)
	// }

	// Create or retrieve a loggers for the given log name
	loggers = loggingClient.Logger("my-logs")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var fileContent ThumbnailFileContent
	if err := json.NewDecoder(r.Body).Decode(&fileContent); err != nil {
		http.Error(w, "Failed to decode message", http.StatusBadRequest)
		return
	}

	loggers.Log(logging.Entry{
		Payload:  "GenerateThumbnail function started",
		Severity: logging.Info,
	})
	log.Println("GenerateThumbnail function started")
	// Decode the image using appropriate format detection
	img, err := FetchAndResizeImage(fileContent.FileURL)
	if err != nil {
		loggers.Log(logging.Entry{
			Payload:  fmt.Sprintf("Error during FetchAndResizeImage: %v", err.Error()),
			Severity: logging.Error,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err.Error())
		return
	} else {
		loggers.Log(logging.Entry{
			Payload:  "Fetch and resize done",
			Severity: logging.Info,
		})
		log.Println("fetch and resize done")

	}

	encoded, err := EncodeImageToJpg(img)
	if err != nil {
		loggers.Log(logging.Entry{
			Payload:  fmt.Sprintf("Error during EncodeImageToJpg: %v", err.Error()),
			Severity: logging.Error,
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err.Error())
		return
	} else {
		loggers.Log(logging.Entry{
			Payload:  "Encoding done",
			Severity: logging.Info,
		})
		log.Println("encoding done")
	}

	uniqueFilename := fmt.Sprintf("%d_%s_%d", time.Now().UnixNano(), "thumbnail", fileContent.DocId)

	object := storageClient.Bucket(bucketName).Object(uniqueFilename)
	thumbnailWC := object.NewWriter(ctx)
	if _, err := io.Copy(thumbnailWC, encoded); err != nil {
		loggers.Log(logging.Entry{
			Payload:  fmt.Sprintf("Error while copying file content to Cloud Storage: %v", err.Error()),
			Severity: logging.Error,
		})
		fmt.Println(err)
		http.Error(w, "Failed to copy file content to Cloud Storage", http.StatusInternalServerError)
		return
	} else {
		loggers.Log(logging.Entry{
			Payload:  "File content copied to Cloud Storage",
			Severity: logging.Info,
		})
		log.Println("File content copied to Cloud Storage")

	}
	if err := thumbnailWC.Close(); err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to close Cloud Storage writer", http.StatusInternalServerError)
		return
	}
	if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to set ACL for Cloud Storage object", http.StatusInternalServerError)
		return
	}
	uploadedFileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, uniqueFilename)
	fmt.Println(uploadedFileURL)
	loggers.Log(logging.Entry{
		Payload:  uploadedFileURL,
		Severity: logging.Info,
	})
	if err := StoreToFirestore(ctx, uploadedFileURL, fileContent.DocId); err != nil {
		loggers.Log(logging.Entry{
			Payload:  fmt.Sprintf("Error while updating Firestore document: %v", err.Error()),
			Severity: logging.Error,
		})
		log.Println(err)
		http.Error(w, "Failed to update Firestore document", http.StatusInternalServerError)
		return
	} else {
		loggers.Log(logging.Entry{
			Payload:  "Thumbnail URL stored in firestore",
			Severity: logging.Info,
		})
		log.Println("Thumbnail URL stored in firestore")
	}
	loggers.Log(logging.Entry{
		Payload:  "Image resized, Thumbnail Created",
		Severity: logging.Info,
	})
	log.Println("Image resized, Thumbnail Created")
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(encoded.Len()))
	w.WriteHeader(http.StatusOK)

}

func EncodeImageToJpg(img *image.Image) (*bytes.Buffer, error) {
	encoded := &bytes.Buffer{}
	err := jpeg.Encode(encoded, *img, nil)
	return encoded, err
}

func FetchAndResizeImage(p string) (*image.Image, error) {
	var dst image.Image

	response, err := http.Get(p)
	if err != nil {
		return &dst, err
	}
	defer response.Body.Close()

	src, _, err := image.Decode(response.Body)
	if err != nil {
		return &dst, err
	}

	dst = imaging.Resize(src, 200, 200, imaging.Lanczos)

	return &dst, nil
}

func StoreToFirestore(ctx context.Context, uploadedFileURL string, docID int) error {
	firestoreClient, _ := firestore.NewClient(ctx, "capstore-takeoff")
	// if err != nil {
	// 	log.Fatalf("Failed to create Firestore client: %v", err)
	// }
	defer firestoreClient.Close()

	docRef := firestoreClient.Collection("Groceries").Doc(strconv.Itoa(docID))

	_, err := docRef.Set(ctx, map[string]interface{}{
		"thumbnailURL": uploadedFileURL,
	}, firestore.MergeAll)
	if err != nil {
		return err
	}
	return nil
}
