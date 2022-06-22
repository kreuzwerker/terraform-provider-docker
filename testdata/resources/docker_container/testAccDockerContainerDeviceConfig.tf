resource "docker_image" "foo" {
  name = "busybox:1.35.0"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest

  devices {
    host_path      = "/dev/zero"
    container_path = "/dev/zero_test"
    permissions    = "rwm"
  }
}
