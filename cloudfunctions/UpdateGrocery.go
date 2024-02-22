package cloudfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"github.com/takeoff-capstone/common"
	"github.com/takeoff-capstone/utils"
)

var (
	Logger *logging.Logger
)

// @Summary Update a grocery item
// @Description Update a grocery item by providing its ID and new data
// @ID update-grocery
// @Accept json
// @Produce json
// @Param id query string true "ID of the grocery item to update"
// @Param json-data formData string true "JSON data containing updated information"
// @Param image formData file false "Image file for the grocery item"
// @Success 201 {object} map[string]interface{} "OK"
// @Failure 400 {string} string "Bad Request: Invalid ID or missing JSON data"
// @Failure 404 {string} string "Not Found: Grocery item not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /UpdateGrocery [patch]
func UpdateGrocery(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST,UPDATE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request..

	w.Header().Set("Access-Control-Allow-Origin", "*")
	// InitLogger(ctx)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Println("Failed to parse multipart form:", err)
		common.RespondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}
	//InfoLog("UpdateGrocery Function started ")
	log.Println("UpdateGrocery Function started ")
	documentID := r.URL.Query().Get("id")
	if documentID == "" {
		//ErrorLog(errors.New("Grocery ID is required"))

		http.Error(w, "Grocery ID is required", http.StatusBadRequest)
		return
	}
	jsonData := r.FormValue("json-data")
	if jsonData == "" {
		log.Print("JSON data is required to create grocery item.")
		common.RespondWithError(w, http.StatusBadRequest, "No 'json-data' field provided in the form")
	}
	formData := make(map[string]interface{})
	if err := json.Unmarshal([]byte(jsonData), &formData); err != nil {
		log.Println("Failed to unmarshal JSON:", err)
		//ErrorLog(err)

		common.RespondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	client, err := utils.CreateFirestoreClient()
	if err != nil {
		//ErrorLog(err)

		http.Error(w, fmt.Sprintf("Failed to create Firestore client: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	docRef := client.Collection("Groceries").Doc(documentID)

	// Check if the document exists
	docSnapshot, err := docRef.Get(ctx)
	if err != nil {

		http.Error(w, fmt.Sprintf("Error getting document: %v", err), http.StatusInternalServerError)
		//ErrorLog(err)

		return
	}

	if !docSnapshot.Exists() {
		http.Error(w, "Document not found", http.StatusNotFound)
		//ErrorLog(err)

		return
	}

	// Unmarshal existing data from the Firestore document
	var existingData map[string]interface{}
	if err := docSnapshot.DataTo(&existingData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal existing document data: %v", err), http.StatusInternalServerError)
		//ErrorLog(err)

		return
	}
	var imageURL string
	file, header, err := r.FormFile("image")
	if err == http.ErrMissingFile {
		// no image provided, proceed without image
		log.Println("No image file")
	} else if err != nil {

		log.Println("Failed to get image file:", err)
		common.RespondWithError(w, http.StatusBadRequest, "Failed to get image file")
		//ErrorLog(err)

		return
	} else {
		storageClient, err := utils.CreateStorageClient()
		if err != nil {
			//ErrorLog(err)

			http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusInternalServerError)
			return
		}
		fileBytes, err := ioutil.ReadAll(file)
		filename := header.Filename
		format := http.DetectContentType(fileBytes)
		if format != "image/jpeg" && format != "image/png" {
			log.Println("Unsupported file format. Only JPG or PNG files are allowed")
			common.RespondWithError(w, http.StatusBadRequest, "Unsupported file format. Only JPG or PNG files are allowed")

			return
		}
		productNameWithoutSpaces := strings.ReplaceAll(filename, " ", "_")

		if imageURLInterface, ok := existingData["image"]; ok && imageURLInterface != nil {
			// Attempt to type assert the value to a string
			if imageURLString, isString := imageURLInterface.(string); isString {
				// The value stored in imageURLString is a string type
				imageURL = imageURLString
			} else {
				http.Error(w, "Image URL is not a string", http.StatusInternalServerError)
			

				return
			}
		} else {
			// Handle case if "image" key does not exist or is nil
			http.Error(w, "Image URL not found in document data", http.StatusNotFound)
			//ErrorLog(err)

			return
		}
		err = common.DeleteImageFromStorage(ctx, imageURL, common.BucketName)
		if err != nil {
			log.Println("Failed to delete image file:", err)
			common.RespondWithError(w, http.StatusInternalServerError, "Failed to read image file")
			//ErrorLog(err)

			return
		}

		// Determine the format of the image based on its header

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			common.RespondWithError(w, http.StatusBadRequest, err.Error())
			//ErrorLog(err)

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
			//ErrorLog(err)

			return
		}
		// Close the writer to finalize the upload
		if err := wc.Close(); err != nil {
			http.Error(w, "Failed to close Cloud Storage writer", http.StatusInternalServerError)
			//ErrorLog(err)

			return
		}
		if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			http.Error(w, "Failed to set ACL for Cloud Storage object", http.StatusInternalServerError)
			//ErrorLog(err)

			return
		}
		uploadedFileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", common.BucketName, uniqueFilename)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			common.RespondWithError(w, http.StatusBadRequest, err.Error())
			//ErrorLog(err)

			return
		}
		id, _ := strconv.Atoi(documentID)

		fmt.Println(uploadedFileURL)
		existingData["image"] = uploadedFileURL
		thumbnail_data := map[string]interface{}{
			"fileURL": uploadedFileURL,
			"ID":      id,
			// Add more audit information as needed
		}
		//err = PublishToPubSub(common.Topic, thumbnail_data)
		//Publish the Thumbnail record to the Pub/Sub topic
		//InfoLog("Thumbnail Published to the Thumbnail_topic successfully")
		log.Println("Thumbnail Published to the Thumbnail_topic successfully")
		err = common.PublishToPubSub(common.Thumbnail_Topic, common.Thumbnail_Topic_subscription, common.Thumbnail_Endpoint, thumbnail_data)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish audit record to Pub/Sub: %v", err), http.StatusInternalServerError)
			//ErrorLog(err)

			return
		}

	}
	productName, ok := existingData["productname"].(string)
	if !ok {
		http.Error(w, "Product name not found in existing data", http.StatusInternalServerError)

		return
	}
	// Merge the existing data with the new data from the form
	for key, value := range formData {
		existingData[key] = value
	}

	// Update the Firestore document with the merged data
	_, err = docRef.Set(ctx, existingData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update document: %v", err), http.StatusInternalServerError)
		//ErrorLog(err)

		return
	}

	auditRecordJSON := map[string]interface{}{
		"Action":      "Update",
		"ID":          documentID,
		"ProductName": productName,
		"Timestamp":   time.Now().Format("2006-01-02 03:04:05 PM"),
	}

	// Publish the audit record to the Pub/Sub topic
	//	err = publishToPubSubAudit_Subscription("Audit-Topic", auditRecordJSON)
	//InfoLog("Audit Published to the Audit_topic successfully")
	log.Println("Audit Published to the Audit_topic successfully")
	err = common.PublishToPubSub(common.Audit_Topic, common.Audit_Topic_subscription, common.Audit_Endpoint, auditRecordJSON)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish audit record to Pub/Sub: %v", err), http.StatusInternalServerError)
		//ErrorLog(err)

		return
	}
	//InfoLog("UpdateGrocery Function completed successfully")
	log.Println("UpdateGrocery Function completed successfully")

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "Document updated successfully", "documentID": "%s"}`, documentID)
}
