resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_container" "foo" {
  image  = docker_image.foo.image_id
  name   = "nginx"
  cpus   = "1.5"
}
