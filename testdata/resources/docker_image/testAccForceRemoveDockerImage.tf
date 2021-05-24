resource "docker_image" "test" {
  name         = "alpine:3.11.5"
  force_remove = true
}
