terraform {
  required_providers {
    bx2cloud = {
      source = "registry.terraform.io/hashicorp/bx2cloud"
    }
  }
}

provider "bx2cloud" {
  host = "localhost:8080"
}

resource "bx2cloud_network" "my_network" {
  internet_access = true
}

resource "bx2cloud_subnetwork" "my_subnetwork" {
  cidr = "10.0.44.0/24"
}
