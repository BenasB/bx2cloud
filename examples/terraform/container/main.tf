terraform {
  required_providers {
    bx2cloud = {
      source = "local/benasb/bx2cloud"
    }
  }
}

provider "bx2cloud" {
  host = "localhost:8080"
}

resource "bx2cloud_network" "my_network" {
  internet_access = false
}

resource "bx2cloud_subnetwork" "my_subnetwork" {
  network_id = bx2cloud_network.my_network.id
  cidr       = "10.0.42.0/24"
}

resource "bx2cloud_container" "my_container" {
  subnetwork_id = bx2cloud_subnetwork.my_subnetwork.id
  image         = "ubuntu:24.04"
  cmd           = ["sleep", "infinity"]
  status        = "running"
}
