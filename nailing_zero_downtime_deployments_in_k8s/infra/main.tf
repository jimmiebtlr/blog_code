variable "project" {}

provider "google" {
  project = var.project
  region  = "us-central1"
}

resource "google_container_cluster" "no_downtime" {
  name               = "no-downtime-deploys"
  location           = "us-central1-a"
  initial_node_count = 1
}

output "kubectl_cfg_cmd" {
  value = "\ngcloud container clusters get-credentials --zone=${google_container_cluster.no_downtime.location} ${google_container_cluster.no_downtime.name}\n"
}