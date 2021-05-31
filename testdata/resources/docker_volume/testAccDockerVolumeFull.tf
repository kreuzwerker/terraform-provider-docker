resource "docker_volume" "foo" {
  name = "testAccDockerVolume_full"

  driver = "local"
  driver_opts = {
    type   = "btrfs"
    device = "/dev/sda2"
  }

  labels {
    label = "com.docker.compose.project"
    value = "test"
  }

  labels {
    label = "com.docker.compose.volume"
    value = "foo"
  }
}
