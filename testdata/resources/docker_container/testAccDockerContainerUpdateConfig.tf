resource "docker_image" "foo" {
  name = "busybox:1.35.0"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest

  restart         = "on-failure"
  max_retry_count = 5
  cpu_shares      = 32
  cpu_set         = "0-1"
  memory          = 512
  memory_swap     = 2048
}
