resource "docker_image" "foo" {
  name = "busybox:1.35.0"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest

  sysctls = {
    "net.ipv4.ip_forward" = "1"
  }
}
