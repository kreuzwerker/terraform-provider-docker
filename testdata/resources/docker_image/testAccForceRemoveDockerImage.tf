resource "docker_image" "test" {
  name         = "alpine:3.14.0"
  force_remove = true
}
