package cloudfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
)

const (
	projectID = "capstore-takeoff"
)

// @Summary Get a grocery item by ID
// @Description Retrieve a grocery item by providing its ID
// @ID get-grocery-by-id
// @Accept json
// @Produce json
// @Param id query int true "ID of the grocery item to retrieve"
// @Success 200 {object} map[string]interface{} "OK"
// @Failure 400 {string} string "Bad Request: Invalid ID"
// @Failure 404 {string} string "Not Found: Grocery item not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/GetGroceryByID [get]
func GetGroceryByID(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request..
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse the grocery ID from the URL parameter
	groceryID := r.URL.Query().Get("id")
	if groceryID == "" {
		http.Error(w, "Grocery ID is required", http.StatusBadRequest)
		log.Println("Grocery ID is required")
		return
	}

	// Convert the groceryID to an integer
	id, err := strconv.Atoi(groceryID)
	if err != nil {
		http.Error(w, "Invalid Grocery ID", http.StatusBadRequest)
		log.Printf("Invalid Grocery ID: %v", err)
		return
	}

	// Log debug information
	log.Printf("Fetching grocery data for ID: %d", id)

	// Retrieve the grocery data from Firestore
	groceryData, err := getGroceryData(ctx, id)
	if err != nil {
		http.Error(w, "Failed to retrieve grocery data", http.StatusInternalServerError)
		log.Printf("Failed to retrieve grocery data: %v", err)
		return
	}

	// Log debug information
	log.Printf("Retrieved grocery data: %v", groceryData)

	// Return the grocery data as JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groceryData)
}

func getGroceryData(ctx context.Context, id int) (map[string]interface{}, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	doc := strconv.Itoa(id)

	// Log debug information
	log.Printf("Fetching Firestore document for ID: %s", doc)

	snapshot, err := client.Collection("Groceries").Doc(doc).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get document from Firestore: %v", err)
	}

	// Extract data from Firestore document snapshot
	var data map[string]interface{}
	if err := snapshot.DataTo(&data); err != nil {
		return nil, fmt.Errorf("failed to convert Firestore data: %v", err)
	}

	// Log debug information
	log.Printf("Fetched data from Firestore: %v", data)

	return data, nil
}
