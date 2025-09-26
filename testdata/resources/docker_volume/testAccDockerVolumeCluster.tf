resource "docker_volume" "foo" {
  name = "testAccDockerVolume_cluster"

  driver = "local"
  driver_opts = {
    type   = "btrfs"
    device = "/dev/sda2"
  }

  cluster {
    scope  = "multi"
    sharing = "all"
    group = "testgroup"

    required_bytes = "1MiB"
    limit_bytes = "2MiB"
  }

}
