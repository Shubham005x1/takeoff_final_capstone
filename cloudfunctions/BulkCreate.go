package cloudfunctions

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/takeoff-capstone/common"
)

const (
	//projectID       = "capstore-takeoff"
	bucketName      = "bulk_data_bucket"
	requiredHeaders = "productname,price,category,weight,brand,itempackagequantity,packageinformation,manufacturer,countryoforigin"
)

// @Summary Bulk upload grocery items
// @Description Uploads multiple grocery items from a CSV or JSON file
// @ID bulk-upload-grocery-items
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV or JSON file containing grocery items"
// @Success 201 {object} map[string]interface{} "File URL sent successfully"
// @Failure 400 {string} string "Bad Request: Please provide a file"
// @Failure 400 {string} string "Bad Request: Unsupported file type. Only CSV or JSON files are allowed"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/bulkUploadGroceryItems [post]
func BulkUploadGroceryItems(w http.ResponseWriter, r *http.Request) {
	//ctx := context.Background()
	// Parse the form data with a max of 10 MB limit for the entire request
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Println("Failed to parse multipart form:", err)
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Validate if the request contains a file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Please provide a file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	fmt.Println("Content Type of the File", contentType)
	if contentType != "text/csv" && contentType != "application/json" {
		http.Error(w, "Unsupported file type. Only CSV or JSON files are allowed", http.StatusBadRequest)
		return
	}
	var uploadedFileURL string
	switch contentType {
	case "text/csv":
		uploadedFileURL, err = readCSVFile(file, w, header)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to Process the csv file : %v", err), http.StatusInternalServerError)
			return

		}
		if uploadedFileURL == "" {
			// Handle the case where there's an error and the URL is empty for CSV
			log.Println("CSV processing failed. URL is empty.")
			http.Error(w, "Failed to process CSV file", http.StatusInternalServerError)
			return
		}

	case "application/json":
		uploadedFileURL, err = readJSONFile(file, w, header)
		if err != nil {
			log.Println(err)
			http.Error(w, fmt.Sprintf("Failed to Process the JSON file : %v", err), http.StatusInternalServerError)
			return

		}
		if uploadedFileURL == "" {
			// Handle the case where there's an error and the URL is empty for JSON
			log.Println("JSON processing failed. URL is empty.")
			http.Error(w, "Failed to process JSON file", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "Unsupported file type. Only CSV or JSON files are allowed", http.StatusBadRequest)
		return
	}

	Bulk_File_Data := map[string]interface{}{
		"fileURL": uploadedFileURL,
	}

	bulk_create_endpoint := "https://us-central1-capstore-takeoff.cloudfunctions.net/download-csv-bulk-upload"

	err = common.PublishToPubSub("Bulk_Create_Topic", "Bulk_Create_Subscription", bulk_create_endpoint, Bulk_File_Data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish audit record to Pub/Sub: %v", err), http.StatusInternalServerError)
		//ErrorLog(err)

		return
	}

	log.Printf("Message: File URL sent successfully. URL: %s", uploadedFileURL)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "File URL sent successfully", "url": "%s"}`, uploadedFileURL)
}

func readCSVFile(file multipart.File, w http.ResponseWriter, header *multipart.FileHeader) (string, error) {
	ctx := context.Background()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read file content: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read file content: %v", err), http.StatusInternalServerError)
		return "", err
	}

	fileReader := strings.NewReader(string(fileContent))

	csvReader := csv.NewReader(fileReader)

	headers, err := csvReader.Read()
	if err != nil {
		log.Printf("Failed to read CSV headers: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read CSV headers: %v", err), http.StatusInternalServerError)
		return "", err
	}

	// Trim whitespaces from headers and check for spaces
	for _, h := range headers {

		if strings.TrimSpace(h) == "" {
			log.Println("CSV file contains an empty header field")
			//http.Error(w, "CSV file contains an empty header field", http.StatusBadRequest)
			return "", fmt.Errorf("CSV file contains an empty header field")
		}
		if strings.Contains(h, " ") {
			log.Printf("CSV file contains a header field with a space: %s", h)
			//http.Error(w, fmt.Sprintf("CSV file contains a header field with a space: %s", h), http.StatusBadRequest)
			return "", fmt.Errorf("CSV file contains a header field with a space: %s", h)
		}
	}

	// Check if all required headers are present in the CSV
	required := strings.Split(requiredHeaders, ",")
	missingHeaders := make([]string, 0)

	// Create a map to track which required headers are found in the CSV
	headerMap := make(map[string]bool)
	for _, h := range headers {
		headerMap[h] = true
	}

	for _, header := range required {
		if !headerMap[header] {
			missingHeaders = append(missingHeaders, header)
		}
	}

	if len(missingHeaders) > 0 {
		log.Printf("CSV is missing required headers: %s", strings.Join(missingHeaders, ","))
		http.Error(w, fmt.Sprintf("CSV is missing required headers: %s", strings.Join(missingHeaders, ",")), http.StatusBadRequest)
		return "", fmt.Errorf("CSV is missing required headers: %s", strings.Join(missingHeaders, ""))
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Printf("Failed to reset file reader position: %v", err)
		http.Error(w, fmt.Sprintf("Failed to reset file reader position: %v", err), http.StatusInternalServerError)
		return "", err
	}

	fileURL, err := storeFile(ctx, file, header)
	if err != nil {
		// Handle the error from storeFile function
		log.Printf("Failed to store the file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to store the file: %v", err), http.StatusInternalServerError)
		return "", err
	}

	return fileURL, nil
}
func readJSONFile(file multipart.File, w http.ResponseWriter, header *multipart.FileHeader) (string, error) {
	ctx := context.Background()
	var groceryItems []map[string]interface{}

	// Decode the JSON data from the file
	err := json.NewDecoder(file).Decode(&groceryItems)
	if err != nil {
		return " ", fmt.Errorf("failed to decode JSON: %v", err)
	}

	requiredFields := []string{"productname", "price", "category", "weight", "brand", "itempackagequantity", "packageinformation", "manufacturer", "countryoforigin"}

	// Check if all required fields are present in each grocery item
	for _, item := range groceryItems {
		for _, field := range requiredFields {
			if _, ok := item[field]; !ok {
				return " ", fmt.Errorf("missing required field '%s' in one or more grocery items", field)
			}
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Printf("Failed to reset file reader position: %v", err)
		http.Error(w, fmt.Sprintf("Failed to reset file reader position: %v", err), http.StatusInternalServerError)
		return " ", err
	}

	fileURL, err := storeFile(ctx, file, header)
	if err != nil {
		return "", fmt.Errorf("failed to store file: %v", err)
	}

	return fileURL, nil
}

func storeFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %v", err)
	}

	if len(fileContent) == 0 {
		return "", fmt.Errorf("file is empty or could not be read")
	}

	fileReader := strings.NewReader(string(fileContent))

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	fileName := header.Filename
	uniqueFilename := fmt.Sprintf("%s_%s", time.Now().Format("20060102"), fileName)
	object := client.Bucket(bucketName).Object(uniqueFilename)
	wc := object.NewWriter(ctx)

	if _, err := io.Copy(wc, fileReader); err != nil {
		wc.Close()
		return "", fmt.Errorf("failed to copy file content to Cloud Storage: %v", err)
	}

	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("failed to close Cloud Storage writer: %v", err)
	}

	if err := object.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return "", fmt.Errorf("failed to set ACL for Cloud Storage object: %v", err)
	}

	uploadedFileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, uniqueFilename)
	return uploadedFileURL, nil
}

// func triggerTheEvent(uploadedFileURL string, docId int, ctx context.Context, w http.ResponseWriter) {
// 	c, err := cloudevents.NewClientHTTP()
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Failed to create CloudEvents client: %v", err), http.StatusInternalServerError)
// 		return
// 	}
// 	event := cloudevents.NewEvent()
// 	event.SetSource("GenerateBulkitem")
// 	event.SetType("file.uploaded")
// 	event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
// 		"fileURL": uploadedFileURL,
// 		"docId":   docId,
// 	})
// 	ctx = cloudevents.ContextWithTarget(ctx, "http://localhost:8087/api/GenerateBulkitem")
// 	//ctx = cloudevents.ContextWithTarget(ctx, "https://us-central1-capstore-takeoff.cloudfunctions.net/Thumbnai_Function")
// 	if result := c.Send(ctx, event); cloudevents.IsUndelivered(result) {
// 		http.Error(w, fmt.Sprintf("Failed to send CloudEvent: %v", result), http.StatusInternalServerError)
// 		return
// 	}
