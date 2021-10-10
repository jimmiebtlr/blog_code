resource "google_container_cluster" "cluster" {
  name     = "prod-cluster"
  location = "us-central1"

  node_locations = ["us-central1-a", "us-central1-b"]

  release_channel {
    channel = "STABLE"
  }

  enable_shielded_nodes = "true"

  workload_identity_config {
    identity_namespace = "${var.project}.svc.id.goog"
  }

  resource_usage_export_config {
    enable_network_egress_metering = false
    enable_resource_consumption_metering = true

    bigquery_destination {
      dataset_id = google_bigquery_dataset.cluster_resource_export.dataset_id
    }
  }

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {
    cluster_ipv4_cidr_block  = "/16"
    services_ipv4_cidr_block = "/22"
  }

  maintenance_policy {
    daily_maintenance_window {
      start_time = "03:00"
    }
  }

  initial_node_count       = 1
  remove_default_node_pool = true

  timeouts {
    create = "20m"
    update = "20m"
  }
}
