resource "docker_image" "this" {
  name = "%s"
}

resource "docker_container" "foo" {
  image   = docker_image.this.name
  name    = "foobar"
  command = ["sh", "-c", "while true ;do wait ;done"]
}
