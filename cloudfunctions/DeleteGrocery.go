package cloudfunctions

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"github.com/takeoff-capstone/common"
	"github.com/takeoff-capstone/utils"
)

var logClient *logging.Client

// @Summary Delete a grocery item
// @Description Delete a grocery item by providing its ID
// @ID delete-grocery
// @Param id query integer true "ID of the grocery item to delete"
// @Success 200 {object} map[string]interface{} "OK"
// @Failure 400 {string} string "Bad Request: Invalid ID"
// @Failure 500 {string} string "Internal Server Error"
// @Router /DeleteGrocery [delete]
func DeleteGrocery(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request..
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ctx := context.Background()
	if logClient == nil {
		if err := initLogging(ctx); err != nil {
			http.Error(w, fmt.Sprintf("Failed to initialize logging: %v", err), http.StatusInternalServerError)
			return
		}
	}
	logger := logClient.Logger("my-log")
	// Extract document ID from the request parameters
	documentIDStr := r.URL.Query().Get("id")
	if documentIDStr == "" {
		http.Error(w, "Document ID is required", http.StatusBadRequest)
		return
	}
	client, err := utils.CreateFirestoreClient()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create Firestore client: %v", err), http.StatusInternalServerError)
		log.Printf("Error creating Firestore client: %v", err)

		return
	}
	defer client.Close()
	documentID, err := strconv.Atoi(documentIDStr)
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		log.Printf("Error converting document ID to int: %v", err)

		return
	}
	log.Printf("Document ID: %s", documentIDStr)

	docRef := client.Collection("Groceries").Doc(documentIDStr)

	// Check if the document exists
	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting document: %v", err), http.StatusInternalServerError)
		log.Printf("Error getting document: %v", err)

		return
	}

	// Read the existing data
	existingData := docSnapshot.Data()
	log.Printf("Existing Data: %+v", existingData)

	productName, ok := existingData["productname"].(string)
	fmt.Println(productName)
	if !ok {
		http.Error(w, "Product name not found in existing data", http.StatusInternalServerError)
		log.Printf("Product name not found in existing data")

		return
	}
	if err := DeleteFromFirestore(ctx, documentID, client); err != nil {
		http.Error(w, "Failed to delete document from Firestore", http.StatusInternalServerError)
		log.Printf("Failed to delete document from Firestore: %v", err)

		return
	}
	log.Printf("Product Name: %s, Found: %t", productName, ok)

	logger.Log(logging.Entry{
		Payload: map[string]interface{}{
			"Action":      "Delete",
			"ID":          documentID,
			"ProductName": productName,
			"Timestamp":   time.Now().Format("2006-01-02 03:04:05 PM"),
			// Add more audit information as needed
		},

		Severity: logging.Info,
	})
	auditRecordJSON := map[string]interface{}{
		"Action":      "Delete",
		"ID":          documentID,
		"ProductName": productName,
		"Timestamp":   time.Now().Format("2006-01-02 03:04:05 PM"),
	}

	// Publish the audit record to the Pub/Sub topic
	//err = publishToPubSub("Audit-Topic", auditRecordJSON)
	log.Println("Audit Published to the Topic")
	err = common.PublishToPubSub(common.Audit_Topic, common.Audit_Topic_subscription, common.Audit_Endpoint, auditRecordJSON)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish audit record to Pub/Sub: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "Document Deleted successfully", "documentID": "%d"}`, documentID)
}

func DeleteFromFirestore(ctx context.Context, documentID int, client *firestore.Client) error {

	doc := strconv.Itoa(documentID)

	// Fetch the document to get the image URL
	docRef := client.Collection("Groceries").Doc(doc)
	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get document: %v", err)
	}

	data := docSnapshot.Data()
	imageURL, ok := data["image"].(string)
	if !ok || imageURL == "" {
		return fmt.Errorf("image URL not found in document")
	}

	// Delete the document from Firestore
	_, err = docRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete document from Firestore: %v", err)
	}

	// Delete the image from Cloud Storage
	// err = deleteImageFromStorage(ctx, imageURL)
	err = common.DeleteImageFromStorage(ctx, imageURL, common.BucketName)
	if err != nil {
		return fmt.Errorf("failed to delete image from Cloud Storage: %v", err)
	}

	return nil
}

func initLogging(ctx context.Context) error {
	var err error
	logClient, err = logging.NewClient(ctx, common.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create logging client: %v", err)
	}
	return nil
}
