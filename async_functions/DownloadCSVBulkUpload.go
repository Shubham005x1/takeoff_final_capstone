package async_functions

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
)

const (
	projectID      = "capstore-takeoff" // Replace with your Firestore project ID
	collectionName = "bulk_data"        // Replace with your Firestore collection name
	logName        = "file-fetch-upload"
)

type FileContent struct {
	FileURL string `json:"fileURL"`
}

var (
	logger *logging.Client
)

func init() {
	// Creates a client.
	var err error
	logger, err = logging.NewClient(context.Background(), projectID)
	if err != nil {
		log.Fatalf("Failed to create logging client: %v", err)
	}
}

func DownloadCSV(w http.ResponseWriter, r *http.Request) {
	var fileContent FileContent
	if err := json.NewDecoder(r.Body).Decode(&fileContent); err != nil {
		http.Error(w, "Failed to decode message", http.StatusBadRequest)
		return
	}
	log.Println("Downloading Csv/JSON and saving Data to the Firestore")
	var isCSV bool

	// Check if the FileURL ends with '.csv' indicating it's a CSV file
	if strings.HasSuffix(strings.ToLower(fileContent.FileURL), ".csv") {
		isCSV = true
	}
	if isCSV {
		// It's CSV data, process it accordingly
		// FetchAndUploadToFirestore function for CSV processing
		if err := FetchAndUploadCSVToFirestore(fileContent.FileURL); err != nil {
			logAndHTTPError(w, http.StatusInternalServerError, "failed to fetch and upload CSV to Firestore", err)
			return
		}
	} else {
		// It's assumed to be JSON data, process it accordingly
		log.Printf("File Content (JSON): %v", fileContent)
		// Log file content to GCP
		logToGCP("File content fetched (JSON): " + fileContent.FileURL)

		if err := FetchAndUploadJSONToFirestore(fileContent.FileURL); err != nil {
			logAndHTTPError(w, http.StatusInternalServerError, "failed to fetch and upload JSON to Firestore", err)
			return
		}
	}
	log.Println("File content fetched and uploaded to Firestore successfully")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File content fetched and uploaded to Firestore successfully")
}

func FetchAndUploadCSVToFirestore(uploadedFileURL string) error {
	// Fetch the file from the URL
	resp, err := http.Get(uploadedFileURL)
	if err != nil {
		return fmt.Errorf("failed to fetch file from URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK response: %v", resp.Status)
	}

	// Read the file content
	csvData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read file content: %v", err)
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Parse CSV data
	reader := csv.NewReader(strings.NewReader(string(csvData)))
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV records: %v", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no records found in CSV data")
	}

	headers := records[0]

	var wg sync.WaitGroup
	for _, record := range records[1:] {
		wg.Add(1)

		// Use a goroutine for parallel processing
		go func(record []string) {
			defer wg.Done()

			// Process record and add to Firestore
			if err := processCSVRecord(ctx, client, collectionName, headers, record); err != nil {
				log.Printf("Error processing record: %v", err)
				// Handle error as needed
			}
		}(record)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return nil

}


func logToGCP(message string) {
	logger.Logger(logName).Log(logging.Entry{Payload: message})
}

func logAndHTTPError(w http.ResponseWriter, statusCode int, message string, err error) {
	log.Printf("%s: %v", message, err)
	logToGCP(fmt.Sprintf("%s: %v", message, err))
	http.Error(w, message, statusCode)
}
func FetchAndUploadJSONToFirestore(jsonFileURL string) error {
	// Fetch the JSON file from the URL
	resp, err := http.Get(jsonFileURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JSON file from URL: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK response while fetching JSON file: %v", resp.Status)
	}

	// Read the file content
	jsonData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JSON file content: %v", err)
	}

	// Initialize Firestore client
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	// Define your Firestore collection name
	collection := client.Collection(collectionName)

	// Unmarshal JSON data
	var items []map[string]interface{}
	if err := json.Unmarshal(jsonData, &items); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %v", err)
	}

	// Upload JSON data to Firestore
	for _, item := range items {
		if _, _, err := collection.Add(ctx, item); err != nil {
			return fmt.Errorf("failed to upload item to Firestore: %v", err)
		}
	}

	return nil
}
func processCSVRecord(ctx context.Context, client *firestore.Client, collectionName string, headers []string, record []string) error {
	item := make(map[string]interface{})

	// Validate and process each column individually
	for i, header := range headers {
		if err := processColumn(header, record[i], item); err != nil {
			log.Printf("Error processing column %s: %v", header, err)
			logToGCP(fmt.Sprintf("Error processing column %s: %v", header, err))

			// If there's an error in any column, skip the entire row
			return nil
		}
	}

	// Add the item to Firestore
	_, _, err := client.Collection(collectionName).Add(ctx, item)
	return err
}

func processColumn(header, value string, item map[string]interface{}) error {
	// Handle special processing for specific columns (e.g., "price")
	switch header {
	case "price":
		// Validate if the "price" column contains a numeric value
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("non-numeric value in 'price' column: %s", value)
		}
		// If it's numeric, add it to the item map
		item[header] = value

	default:
		// For other columns, add them to the item map as-is
		item[header] = value
	}

	return nil
}
