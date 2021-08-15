resource "docker_image" "test" {
  name         = "alpine:3.14.1"
  force_remove = true
}
