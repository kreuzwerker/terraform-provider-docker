resource "docker_image" "foo" {
  name         = "busybox:1.35.0"
  keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest

  healthcheck {
    test         = ["CMD", "/bin/true"]
    interval     = "30s"
    timeout      = "5s"
    start_period = "15s"
    retries      = 10
  }
}
