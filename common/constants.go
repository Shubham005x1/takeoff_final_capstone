package common

import "net/http"

const (
	ProjectID                    = "capstore-takeoff"
	BucketName                   = "groceries_images"
	LogName                      = "create-grocery-log"
	Thumbnail_Topic_subscription = "Thumbnail_Subscription"
	Thumbnail_Topic              = "Thumbnail_topic"
	Thumbnail_Endpoint           = "https://us-central1-capstore-takeoff.cloudfunctions.net/thumbnail-generation"
)

const (
	Audit_Topic              = "Audit-Topic"
	Audit_Topic_subscription = "Audit_Subscription"
	Audit_Endpoint           = "https://us-central1-capstore-takeoff.cloudfunctions.net/auditlog-generation"
)

type PubSubMessage struct {
	Action      string `json:"action"`
	ID          string `json:"id"`
	ProductName string `json:"product_name"`
	Timestamp   string `json:"timestamp"`
}

func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}
