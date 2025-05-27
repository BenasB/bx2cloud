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

resource "bx2cloud_network" "my-net" {
  internet_access = false
}

resource "bx2cloud_subnetwork" "my-subnet" {
  cidr = "10.0.42.0/24"
}
