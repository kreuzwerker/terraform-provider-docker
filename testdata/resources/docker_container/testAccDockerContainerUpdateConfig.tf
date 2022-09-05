resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id

  restart         = "on-failure"
  max_retry_count = 5
  cpu_shares      = 32
  cpu_set         = "0-1"
  memory          = 512
  memory_swap     = 2048
}
