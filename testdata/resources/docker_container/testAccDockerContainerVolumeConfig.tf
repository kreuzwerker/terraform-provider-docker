resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_volume" "foo" {
  name = "testAccDockerContainerVolume_volume"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id

  volumes {
    volume_name    = docker_volume.foo.name
    container_path = "/tmp/volume"
    read_only      = false
  }
}
