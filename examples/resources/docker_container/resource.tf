# Start a container
resource "docker_container" "ubuntu" {
  name  = "foo"
  image = docker_image.ubuntu.latest
}

# Find the latest Ubuntu precise image.
resource "docker_image" "ubuntu" {
  name = "ubuntu:jammy"
}
