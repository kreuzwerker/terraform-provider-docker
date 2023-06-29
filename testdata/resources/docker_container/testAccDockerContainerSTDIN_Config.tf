resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_container" "foo" {
  name       = "tf-test"
  image      = docker_image.foo.image_id
  stdin_open = true
}
