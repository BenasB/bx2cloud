resource "bx2cloud_container" "my_container" {
  subnetwork_id = bx2cloud_subnetwork.my_subnetwork.id
  image         = "ubuntu:24.04"
  cmd           = ["sleep", "infinity"]
  status        = "running"
  env = {
    FOO = "bar"
  }
}
