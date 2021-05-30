resource "docker_network" "foo" {
  name    = "bar"
  driver  = "overlay"
  ingress = true
}
