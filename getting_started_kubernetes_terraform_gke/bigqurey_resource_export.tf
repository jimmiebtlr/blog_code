resource "google_bigquery_dataset" "cluster_resource_export" {
  dataset_id                  = "cluster_resource_export"
  friendly_name               = "Cluster Resource Export"
  location                    = "US"

  access {
    role          = "OWNER"
    user_by_email = google_service_account.bqowner.email
  }

  access {
    role   = "READER"
    domain = "hashicorp.com"
  }
}

resource "google_service_account" "bqowner" {
  account_id = "bqowner"
}
