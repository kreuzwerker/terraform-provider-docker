resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name     = "tf-test"
  image    = docker_image.foo.id
  start    = false
  must_run = false
}
