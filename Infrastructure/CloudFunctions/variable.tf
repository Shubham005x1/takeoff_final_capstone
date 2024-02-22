variable "project_id" {
  default = "capstore-takeoff"
}

variable "region" {
  default = "us-central1"
}

variable "functions" {
  type = map(object({
    zip        = string
    name       = string
    trigger    = string
    runtime    = string
    entrypoint = string
      iam_member = string
  }))

  default = {
    "creategrocery" : {
      zip        = "CreateGrocery.zip"
      name       = "creategrocery"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "CreateGrocery"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    }
    "login" : {
      zip        = "login.zip"
      name       = "login"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "Login"
      iam_member   = "allUsers"

    }
    "deletegrocery" : {
      zip        = "DeleteGrocery.zip"
      name       = "deletegrocery"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "DeleteGrocery"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    },
    "viewallgrocery" : {
      zip        = "ViewAllGrocery.zip"
      name       = "viewallgrocery"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "ViewAllGroceries"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    },
    "viewgrocerybyid" : {
      zip        = "ViewGroceryById.zip"
      name       = "viewgrocerybyid"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "GetGroceryByID"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    },
    "updategrocery" : {
      zip        = "UpdateGrocery.zip"
      name       = "updategrocery"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "UpdateGrocery"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    },
    "bulkcreate" : {
      zip        = "BulkCreate.zip"
      name       = "bulkcreate"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "BulkUploadGroceryItems"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    }
    "auditlog-generation" : {
      zip        = "AuditLog_Generation.zip"
      name       = "auditlog-generation"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "ProcessPubSubMessages"
      iam_member = "serviceAccount:pubsub-pushsubscription@capstore-takeoff.iam.gserviceaccount.com"
    }
     "creategroceries" : {
      zip        = "create.zip"
      name       = "creategroceries"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "CreateGrocery"
      iam_member   = "serviceAccount:api-gateway-service-acc@capstore-takeoff.iam.gserviceaccount.com"

    }
    "thumbnail-generation" : {
      zip        = "Thumbnail_Generation.zip"
      name       = "thumbnail-generation"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "GenerateThumbnail"
      iam_member = "serviceAccount:pubsub-pushsubscription@capstore-takeoff.iam.gserviceaccount.com"
    }
    "download-csv-bulk-upload" : {
      zip        = "DownloadCSVBulkUpload.zip"
      name       = "download-csv-bulk-upload"
      trigger    = "http-trigger"
      runtime    = "go121"
      entrypoint = "DownloadCSV"
     iam_member = "serviceAccount:pubsub-pushsubscription@capstore-takeoff.iam.gserviceaccount.com"

    }
   }
}