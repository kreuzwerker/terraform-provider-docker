resource "docker_image" "test" {
  name         = "alpine:3.13.5"
  force_remove = true
}
