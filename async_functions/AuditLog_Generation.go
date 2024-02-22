package async_functions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
)

// ProcessPubSubMessages is an HTTP handler that processes Pub/Sub push messages.
func ProcessPubSubMessages(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Failed to decode message", http.StatusBadRequest)
		return
	}
	log.Printf("Received message data: %+v", data)

	if err := storeInFirestore(ctx, data); err != nil {
		log.Printf("Failed to store message in Firestore: %v", err)
		http.Error(w, "Failed to process message", http.StatusInternalServerError)
		return
	}
	log.Print("Audit Log Added Successfully")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "Audit Log Added Successfully")
}

// Function to store the received Pub/Sub message data in Firestore
func storeInFirestore(ctx context.Context, data map[string]interface{}) error {
	projectID := "capstore-takeoff" // Replace with your GCP project ID

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	auditCol := client.Collection("Audit_Logs")

	if _, _, err := auditCol.Add(ctx, data); err != nil {
		return fmt.Errorf("failed to add audit record to Firestore: %v", err)
	}

	return nil
}
