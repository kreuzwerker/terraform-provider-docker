resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.id

  healthcheck {
    test         = ["CMD", "/bin/true"]
    interval     = "30s"
    timeout      = "5s"
    start_period = "15s"
    retries      = 10
  }
}
