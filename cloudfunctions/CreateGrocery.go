// SECOND VERSION
package cloudfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/takeoff-capstone/common"
	"github.com/takeoff-capstone/utils"
	"github.com/takeoff-capstone/validations"
	"google.golang.org/api/iterator"
)

func init() {
	functions.HTTP("CreateGrocery", CreateGrocery)
}

// @Summary Create a new grocery item
// @Description Create a new grocery item with the provided data and image
// @ID create-grocery
// @Accept json
// @Produce json
// @Param json-data formData string true "JSON data for the grocery item"
// @Param image formData file true "Image file for the grocery item"
// @Success 201 {object} string "File uploaded successfully"
// @Failure 400 {object} string "Bad Request: Invalid JSON payload or missing required fields"
// @Failure 500 {object} string "Internal Server Error"
// @Router /CreateGrocery [post]
func CreateGrocery(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request..
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Println("Failed to parse multipart form:", err)
		common.RespondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}
	ctx := context.Background()
	//loggingClient, err := logging.NewClient(ctx, common.ProjectID)
	// if err != nil {
	// 	log.Fatalf("Failed to create logging client: %v", err)
	// }

	// logger := loggingClient.Logger(common.LogName)
	storageClient, _ := utils.CreateStorageClient()
	log.Println("Started processing request")

	jsonData := r.FormValue("json-data")
	if jsonData == "" {
		log.Print("JSON data is required to create grocery item.")
		common.RespondWithError(w, http.StatusBadRequest, "No 'json-data' field provided in the form")
	}
	formData := make(map[string]interface{})
	if err := json.Unmarshal([]byte(jsonData), &formData); err != nil {
		log.Println("Failed to unmarshal JSON:", err)
		common.RespondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	documentID := generateUniqueID()
	formData["id"] = documentID

	requiredKeys := map[string]bool{
		"productname":         true,
		"price":               true,
		"category":            true,
		"weight":              true,
		"brand":               true,
		"itempackagequantity": true,
		"packageinformation":  true,
		"manufacturer":        true,
		"countryoforigin":     true,
	}
	missingFields := []string{} // Slice to store missing fields

	// Check if all required keys are present in formData
	for key := range requiredKeys {
		if _, ok := formData[key]; !ok {
			missingFields = append(missingFields, key) // Store missing field
		}
	}
	if len(missingFields) > 0 {
		// Respond with a message listing all missing fields
		missingFieldsMessage := fmt.Sprintf("Fields ['%s'] are required", strings.Join(missingFields, "', '"))
		http.Error(w, missingFieldsMessage, http.StatusBadRequest)
		return
	}
	var productName string
	productNameRaw, ok := formData["productname"]
	if !ok {
		http.Error(w, "Product name field is missing", http.StatusBadRequest)
		return
	}

	productName, ok = productNameRaw.(string)
	if !ok {
		http.Error(w, "Invalid type for product name", http.StatusBadRequest)
		return
	}

	// Check if the product name already exists in the database
	if exists, err := checkDuplicateProduct(ctx, productName); err != nil {
		http.Error(w, fmt.Sprintf("Error checking duplicate product: %v", err), http.StatusInternalServerError)
		return
	} else if exists {
		log.Println("Duplicate product found")

		http.Error(w, "Duplicate product found", http.StatusBadRequest)
		return
	}
	priceStr, ok := formData["price"]
	if !ok {
		http.Error(w, "Price field is missing", http.StatusBadRequest)
		return
	}

	var price string
	switch v := priceStr.(type) {
	case string:
		price = v
	default:
		// Convert the price to string (assuming it's a numeric value)
		price = fmt.Sprintf("%v", v)
	}

	validatedPrice, _ := validations.ValidatePrice(price)

	formData["price"] = validatedPrice
	itemPackageQuantityStr, ok := formData["itempackagequantity"]
	if !ok {
		http.Error(w, "Item package quantity field is missing", http.StatusBadRequest)
		return
	}

	var itemPackageQuans string
	switch v := itemPackageQuantityStr.(type) {
	case string:
		itemPackageQuans = v
	default:
		// Convert the item package quantity to string (assuming it's a numeric value)
		itemPackageQuans = fmt.Sprintf("%v", v)
	}

	validatedItemPackageQuantity, err := validations.ValidateItemPackageQuantity(itemPackageQuans)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	formData["itempackagequantity"] = validatedItemPackageQuantity
	file, header, err := r.FormFile("image")
	// log.Printf("Original image format: %s", formatimg)
	var uploadedFileURL string
	if err == http.ErrMissingFile {
		// no image provided, proceed without image
		common.RespondWithError(w, http.StatusBadRequest, "Image is required")
		log.Println("No image file")
		return

	} else if err != nil {
		log.Println("Failed to get image file:", err)
		common.RespondWithError(w, http.StatusBadRequest, "Failed to get image file")
		return
	} else {
		fileBytes, err := ioutil.ReadAll(file)
		filename := header.Filename

		if err != nil {
			log.Println("Failed to read image file:", err)
			common.RespondWithError(w, http.StatusInternalServerError, "Failed to read image file")
			return
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusInternalServerError)
			return
		}
		// Determine the format of the image based on its header
		format := http.DetectContentType(fileBytes)
		if format != "image/jpeg" && format != "image/png" {
			log.Println("Unsupported file format. Only JPG or PNG files are allowed")
			common.RespondWithError(w, http.StatusBadRequest, "Unsupported file format. Only JPG or PNG files are allowed")
			return
		}
		productNameWithoutSpaces := strings.ReplaceAll(filename, " ", "_")
		log.Println(format)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			common.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		// Create a unique filename for the uploaded file
		uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), productNameWithoutSpaces)

		// Set up the destination object in the Cloud Storage bucket
		object := storageClient.Bucket(common.BucketName).Object(uniqueFilename)

		// Create a writer to the object
		wc := object.NewWriter(ctx)
		// Copy the file content to the Cloud Storage object
		if _, err := io.Copy(wc, file); err != nil {
			http.Error(w, "Failed to copy file content to Cloud Storage", http.StatusInternalServerError)
			return
		}
		// Close the writer to finalize the upload
		if err := wc.Close(); err != nil {
			http.Error(w, "Failed to close Cloud Storage writer", http.StatusInternalServerError)
			return
		}
		if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			http.Error(w, "Failed to set ACL for Cloud Storage object", http.StatusInternalServerError)
			return
		}
		uploadedFileURL = fmt.Sprintf("https://storage.googleapis.com/%s/%s", common.BucketName, uniqueFilename)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			common.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	thumbnail_data := map[string]interface{}{
		"fileURL": uploadedFileURL,
		"ID":      documentID,
		// Add more audit information as needed
	}
	//Thumbnail Publish
	//triggerTheEvent(uploadedFileURL, documentID, ctx, w)
	err = common.PublishToPubSub(common.Thumbnail_Topic, common.Thumbnail_Topic_subscription, common.Thumbnail_Endpoint, thumbnail_data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish audit record to Pub/Sub: %v", err), http.StatusInternalServerError)
		return
	}
	formData["image"] = uploadedFileURL
	if err := saveToFirestore(ctx, documentID, formData); err != nil {
		log.Println("Failed to save data to Firestore")
		http.Error(w, "Failed to save data to Firestore", http.StatusInternalServerError)
		return

	}
	// logger.Log(logging.Entry{
	// 	Payload: map[string]interface{}{
	// 		"message": "Completed processing request",
	// 		"method":  r.Method,
	// 		"url":     r.URL.String(),
	// 	},
	// 	Severity: logging.Info,
	// })
	log.Println("Completed processing request")
	fmt.Println(formData)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "File uploaded successfully", "url": "%s"}`, uploadedFileURL)

}
func saveToFirestore(ctx context.Context, documentID int, data map[string]interface{}) error {
	client, err := utils.CreateFirestoreClient()
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()
	doc := strconv.Itoa(documentID)
	_, err = client.Collection("Groceries").Doc(doc).Set(ctx, data)

	if err != nil {
		return fmt.Errorf("failed to add document to Firestore: %v", err)
	}

	return nil
}

func generateUniqueID() int {
	rand.Seed(time.Now().UnixNano())

	const max = 999999 // Maximum 6-digit number

	// Generate a random number until it's within the range of 6 digits
	return rand.Intn(max + 1)
}

func checkDuplicateProduct(ctx context.Context, productName string) (bool, error) {
	client, err := utils.CreateFirestoreClient()
	if err != nil {
		return false, fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Query the database to check for duplicate product name
	iter := client.Collection("Groceries").Where("productname", "==", productName).Documents(ctx)
	defer iter.Stop()

	// Check if there are any documents with the same product name
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false, fmt.Errorf("error iterating over query results: %v", err)
		}

		// If there is at least one document, it means the product name is a duplicate
		return true, nil
	}

	return false, nil
}

// func triggerTheEvent(uploadedFileURL string, docId int, ctx context.Context, w http.ResponseWriter) {
// 	c, err := cloudevents.NewClientHTTP()
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to create CloudEvents client: %v", err), http.StatusInternalServerError)
// 		return
// 	}
// 	event := cloudevents.NewEvent()
// 	event.SetSource("UploadGroceryItems")
// 	event.SetType("file.uploaded")
// 	event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
// 		"fileURL": uploadedFileURL,
// 		"docId":   docId,
// 	})
// 	ctx = cloudevents.ContextWithTarget(ctx, "http://localhost:8087/api/GenerateThumbnail")
// 	//ctx = cloudevents.ContextWithTarget(ctx, "https://us-central1-capstore-takeoff.cloudfunctions.net/Thumbnai_Function")
// 	if result := c.Send(ctx, event); cloudevents.IsUndelivered(result) {
// 		http.Error(w, fmt.Sprintf("Failed to send CloudEvent: %v", result), http.StatusInternalServerError)
// 		return
// 	}

// }
