terraform {
  required_providers {
    bx2cloud = {
      source = "registry.terraform.io/hashicorp/bx2cloud"
    }
  }
}

provider "bx2cloud" {}

data "bx2cloud_vpc" "example" {}
