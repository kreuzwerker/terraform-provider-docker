resource "docker_image" "foo" {
  name         = "busybox:1.35.0"
  keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest

  ports {
    internal = 80
  }

  ports {
    internal = 81
  }
}
