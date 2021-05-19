resource "docker_container" "foo" {
  name  = "foo"
  image = "nginx"

  ports {
    internal = "80"
    external = "8080"
  }
}
