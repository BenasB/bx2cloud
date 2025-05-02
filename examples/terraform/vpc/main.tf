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

resource "bx2cloud_vpc" "my-vpc" {
  name = "my-tf-vpc"
  cidr = "10.0.4.0/24"
}
