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

data "bx2cloud_network" "first" {
  id = 1
}

output "test_output" {
  value = data.bx2cloud_network.first
}
