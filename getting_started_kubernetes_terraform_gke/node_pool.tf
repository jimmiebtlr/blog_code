resource "google_container_node_pool" "primary" {
  name     = "primary"

  location = "us-central1"
  node_locations = ["us-central1-a", "us-central1-b"]

  cluster  = google_container_cluster.cluster.name

  initial_node_count = 1
  autoscaling {
    min_node_count = 1
    max_node_count = 3
  }

  node_config {
    preemptible  = false
    machine_type = "e2-medium"

    workload_metadata_config {
      node_metadata = "GKE_METADATA_SERVER"
    }

    metadata = {
      disable-legacy-endpoints = true
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/cloud-platform",
    ]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  timeouts {
    create = "20m"
    update = "20m"
  }
}

