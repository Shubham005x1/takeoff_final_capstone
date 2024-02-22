# variables.tf

variable "topics_and_subscriptions" {
  type = map(object({
    topic_name        = string
    subscription_name = string
    push_endpoint = string
    ack_deadline_seconds = string 
  }))
  default = {
    "Audit-Topic" : {
      topic_name        = "Audit-Topic"
      subscription_name = "Audit_Subscription"
      push_endpoint = "https://us-central1-capstore-takeoff.cloudfunctions.net/auditlog-generation"
      ack_deadline_seconds = 20

    },
    "Bulk_Create_Topic" : {
      topic_name        = "Bulk_Create_Topic"
      subscription_name = "Bulk_Create_Subscription"
      push_endpoint = "https://us-central1-capstore-takeoff.cloudfunctions.net/download-csv-bulk-upload"
      ack_deadline_seconds = 60
    },
    "Thumbnail_topic" : {
      topic_name        = "Thumbnail_topic"
      subscription_name = "Thumbnail_Subscription"
       push_endpoint = "https://us-central1-capstore-takeoff.cloudfunctions.net/thumbnail-generation"
       ack_deadline_seconds = 30
    }
  }
}
