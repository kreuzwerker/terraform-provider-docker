# Creates a docker volume "shared_volume".
resource "docker_volume" "shared_volume" {
  name = "shared_volume"
}

# Reference the volume with ${docker_volume.shared_volume.name}