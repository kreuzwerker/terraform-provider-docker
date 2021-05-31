resource "docker_image" "foobarzoo" {
  name         = "crux:3.1"
  keep_locally = true
}
