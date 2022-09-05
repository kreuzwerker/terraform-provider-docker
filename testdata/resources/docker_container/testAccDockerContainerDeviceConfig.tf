resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id

  devices {
    host_path      = "/dev/zero"
    container_path = "/dev/zero_test"
    permissions    = "rwm"
  }
}
