---
sidebar_position: 1
---

# Getting started

Firstly, [install](./installation.md) the bx2cloud Terraform provider.

Make sure you have a running [bx2cloud API](../api/installation.md) instance.

Then, you can create the starting `main.tf` file

```hcl title="main.tf"
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
```

And run `terraform init`, which should result in output similar to the following:

```sh
$ terraform init
Initializing the backend...
Initializing provider plugins...
- Finding local/benasb/bx2cloud versions matching "0.1.0"...
- Installing local/benasb/bx2cloud v0.1.0...
- Installed local/benasb/bx2cloud v0.1.0 (unauthenticated)
...
```

With that, you are ready to start creating bx2cloud resources through Terraform. In your `main.tf`, you can add:

```hcl title="main.tf"
resource "bx2cloud_network" "my_network" {
  internet_access = false
}

resource "bx2cloud_subnetwork" "my_subnetwork" {
  network_id = bx2cloud_network.my_network.id
  cidr       = "10.0.42.0/24"
}

resource "bx2cloud_container" "my_container" {
  subnetwork_id = bx2cloud_subnetwork.my_subnetwork.id
  image         = "nginx:latest"
}
```

And run `terraform apply`:

```sh
$ terraform apply

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # bx2cloud_container.my_container will be created
  ...

  # bx2cloud_network.my_network will be created
  ...

  # bx2cloud_subnetwork.my_subnetwork will be created
  ...

Plan: 3 to add, 0 to change, 0 to destroy.

...

bx2cloud_network.my_network: Creating...
bx2cloud_network.my_network: Creation complete after 0s [id=4]
bx2cloud_subnetwork.my_subnetwork: Creating...
bx2cloud_subnetwork.my_subnetwork: Creation complete after 0s [id=4]
bx2cloud_container.my_container: Creating...
bx2cloud_container.my_container: Creation complete after 4s [id=1]

Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
```
