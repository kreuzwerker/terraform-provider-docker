resource "docker_image" "foobarzoo" {
  name         = "busybox:latest"
  keep_locally = true
}
