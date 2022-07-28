resource "docker_image" "ubuntu" {
  name = "%s"
}

resource "docker_container" "foo" {
  depends_on = [
    docker_image.ubuntu
  ]
  image   = docker_image.ubuntu.latest
  name    = "foobar"
  command = ["sh", "-c", "while true ;do wait ;done"]
}
