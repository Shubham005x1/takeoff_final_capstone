provider "google" {
  credentials = file("../terraform123.json")
  project     = "capstore-takeoff"
  region      = "us-central1"  # Change to your desired region
}
