package cloudfunctions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const (
	//projectID = "capstore-takeoff"
	pageSize = 4 // Define your page size here
)

// @Summary View all groceries
// @Description Retrieve a list of groceries with optional filters and pagination.
// @ID view-all-groceries
// @Accept json
// @Produce json
// @Param pageToken query string false "Page token for cursor-based pagination"
// @Param productname query string false "Filter by product name"
// @Param priceFilter query string false "Price filter format: 'gt:100', 'eq:50', 'lt:200'"
// @Param category query string false "Filter by category"
// @Success 200 {object} map[string]interface{} "OK"
// @Router /ViewAllGroceries [get]
func ViewAllGroceries(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Set CORS headers for the main request.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "capstore-takeoff")
	if err != nil {
		log.Printf("Failed to create Firestore client: %v\n", err)

		http.Error(w, fmt.Sprintf("Failed to create Firestore client: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// Get the page token from the query parameter
	pageToken := r.URL.Query().Get("pageToken")
	productname := r.URL.Query().Get("productname")
	priceFilter := r.URL.Query().Get("priceFilter") // Format: "gt:100", "eq:50", "lt:200"
	category := r.URL.Query().Get("category")
	// var filters []firestore.Query
	log.Printf("Received request with parameters - pageToken: %s, productname: %s, priceFilter: %s, category: %s\n", pageToken, productname, priceFilter, category)

	// Set up the initial query with a page size
	query := client.Collection("Groceries").OrderBy("id", firestore.Asc).Limit(pageSize)
	priceFilterQuery := client.Collection("Groceries").OrderBy("price", firestore.Asc).Limit(pageSize)
	if productname != "" {
		query = query.Where("productname", "==", productname)
		log.Printf("Added productname filter: %s\n", productname)

	}
	if priceFilter != "" {
		components := strings.Split(priceFilter, ":")
		if len(components) != 2 {
			log.Printf("Invalid priceFilter format. Use 'gt', 'eq', or 'lt' with a number.\n")

			http.Error(w, "Invalid priceFilter format. Use 'gt', 'eq', or 'lt' with a number.", http.StatusBadRequest)
			return
		}
		filterType := components[0]
		priceValue, err := strconv.Atoi(components[1])
		if err != nil {
			log.Printf("Invalid price value provided: %v\n", err)

			http.Error(w, "Invalid price value provided.", http.StatusBadRequest)
			return
		}
		switch filterType {
		case "gt":
			query = priceFilterQuery.Where("price", ">", priceValue)
			log.Printf("Added price filter: price > %d\n", priceValue)
		case "eq":
			query = priceFilterQuery.Where("price", "==", priceValue)
			log.Printf("Added price filter: price == %d\n", priceValue)
		case "lt":
			query = priceFilterQuery.Where("price", "<", priceValue)
			log.Printf("Added price filter: price < %d\n", priceValue)
		default:
			log.Printf("Invalid priceFilter type. Use 'gt', 'eq', or 'lt'.\n")
			http.Error(w, "Invalid priceFilter type. Use 'gt', 'eq', or 'lt'.", http.StatusBadRequest)
			return
		}
	}

	if category != "" {
		query = query.Where("category", "==", category)
		log.Printf("Added category filter: %s\n", category)
	}

	// If a page token is provided, use it for cursor-based pagination
	if pageToken != "" {
		decodedToken, err := base64.StdEncoding.DecodeString(pageToken)
		if err != nil {
			log.Printf("Invalid pageToken provided: %v\n", err)
			http.Error(w, fmt.Sprintf("Invalid pageToken provided: %v", err), http.StatusBadRequest)
			return
		}

		tokenString := string(decodedToken)

		// Convert the string token to int64
		cursorInt, err := strconv.ParseInt(tokenString, 10, 64)
		if err != nil {
			log.Printf("Error converting token to int64: %v\n", err)
			http.Error(w, fmt.Sprintf("Error converting token to int64: %v", err), http.StatusInternalServerError)
			return
		}

		// Use the int64 cursor for the query StartAfter
		query = query.StartAfter(cursorInt)
		log.Printf("Using pageToken for cursor-based pagination: %s\n", pageToken)
	}
	// Fetch documents based on the query
	docs := query.Documents(ctx)
	var groceries []map[string]interface{}

	// Iterate through fetched documents
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Failed to iterate over groceries: %v\n", err)

			http.Error(w, fmt.Sprintf("Failed to iterate over groceries: %v", err), http.StatusInternalServerError)
			return
		}
		grocery := doc.Data()
		groceries = append(groceries, grocery)
	}
	fmt.Println("Fetched groceries:", groceries)
	// Get the last document for the next page token
	nextPageToken := ""
	if len(groceries) == pageSize {
		lastDoc := groceries[len(groceries)-1]
		lastDocID, ok := lastDoc["id"].(int64) // Assuming "id" is an int field
		if !ok {
			http.Error(w, "Failed to convert lastDocID to int", http.StatusInternalServerError)
			return
		}
		// Convert integer ID to string and encode it
		idString := fmt.Sprintf("%d", lastDocID)
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(idString))
	}

	// Prepare the URL for the next page with the nextPageToken as a query parameter
	//	nextPageURL := fmt.Sprintf("https://us-central1-capstore-takeoff.cloudfunctions.net/ViewAllGroceryNoPagination?pageToken=%s", nextPageToken)
	//nextPageURL := fmt.Sprintf("http://localhost:8084/api/ViewAllGroceries?pageToken=%s", nextPageToken)

	// Prepare the response with groceries, nextPageToken, and nextPageURL
	response := map[string]interface{}{
		"groceries":      groceries,
		"nextPageToken":  nextPageToken,
		"pageTokenParam": "pageToken",
	}

	// Encode response as JSON and set content type
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode groceries: %v", err), http.StatusInternalServerError)
		return
	}
}
