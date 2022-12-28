resource "docker_image" "test" {
  name         = "alpine:3.16.0"
  force_remove = true
}
