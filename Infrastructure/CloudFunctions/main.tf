resource "google_storage_bucket" "bucket" {
  project  = var.project_id
  name     = "${var.project_id}-test-terraform"
  location = var.region
}


resource "google_storage_bucket_object" "function_zip" {
  for_each = var.functions
  name     = each.key
  bucket   = google_storage_bucket.bucket.name
  # source   = data.archive_file.function_src[each.key].output_path
  source = each.value.zip
}


resource "google_cloudfunctions2_function" "function" {
  for_each = var.functions
  name     = each.value.name
  location = var.region

  build_config {
    runtime     = each.value.runtime
    entry_point = each.value.entrypoint

    source {
      storage_source {
        bucket = google_storage_bucket.bucket.name
        object = google_storage_bucket_object.function_zip[each.key].name
      }
    }
  }
  service_config {
    min_instance_count             = 1
    available_memory               = "256Mi"
    timeout_seconds                = 120
    all_traffic_on_latest_revision = false
    service_account_email          = "capstone-takeoff@capstore-takeoff.iam.gserviceaccount.com"
  }
  #  lifecycle {
  #   prevent_destroy = true
  # }
}

resource "google_cloud_run_service_iam_member" "member" {
  for_each = var.functions

  location = google_cloudfunctions2_function.function[each.key].location
  service  = each.key
  role     = "roles/run.invoker"
   member   = each.value.iam_member
//member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"
  #  member = "allUsers"
}