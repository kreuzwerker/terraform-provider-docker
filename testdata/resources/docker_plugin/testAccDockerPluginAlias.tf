resource "docker_plugin" "test" {
  name          = "docker.io/tiborvass/sample-volume-plugin:latest"
  alias         = "sample:latest"
  force_destroy = true
}
