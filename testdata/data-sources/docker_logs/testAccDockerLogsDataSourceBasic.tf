
resource "docker_container" "busybox" {
  name = "busybox"
  image = "busybox"
  command = ["sh", "-c", "echo Hello World && echo Hello World && echo Hello World"]
  must_run = false
}

data "docker_logs" "logs_basic" {
  name = docker_container.busybox.id
}

data "docker_logs" "logs_discard_headers_false" {
  name = docker_container.busybox.id
  discard_headers = false
}

