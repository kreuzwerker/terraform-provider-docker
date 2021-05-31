resource "docker_plugin" "test" {
  name          = "docker.io/tiborvass/sample-volume-plugin:latest"
  force_destroy = true
}
