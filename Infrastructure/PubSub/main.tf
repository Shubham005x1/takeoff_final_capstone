
resource "google_pubsub_topic" "topics" {
  for_each = var.topics_and_subscriptions

  name = each.value.topic_name
}

resource "google_pubsub_subscription" "subscriptions" {
  for_each = var.topics_and_subscriptions

  name  = each.value.subscription_name
  topic = google_pubsub_topic.topics[each.key].id

  ack_deadline_seconds = each.value.ack_deadline_seconds

  push_config {
    push_endpoint = each.value.push_endpoint

    no_wrapper {
      write_metadata = false
    }

    oidc_token {
      service_account_email = "pubsub-pushsubscription@capstore-takeoff.iam.gserviceaccount.com"
    }
  }
}
