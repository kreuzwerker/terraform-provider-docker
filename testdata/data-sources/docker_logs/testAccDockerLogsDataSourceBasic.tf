
resource "docker_container" "busybox" {
  name = "busybox"
  image = "busybox"
  command = ["sh", "-c", "echo Hello World && echo Hello World && echo Hello World"]
  must_run = false
}

data "docker_logs" "logs" {
  name = docker_container.busybox.id
}
