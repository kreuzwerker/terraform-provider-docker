resource "docker_image" "foobarzoo" {
  name         = "busybox:1.35.0"
  keep_locally = true
}
