resource "docker_image" "test" {
  name         = "alpine:3.14.2"
  force_remove = true
}
