resource "docker_image" "foo_mounts" {
  name = "busybox:1.35.0"
}

resource "docker_volume" "foo_mounts" {
  name = "testAccDockerContainerMounts_volume"
}

resource "docker_container" "foo_mounts" {
  name  = "tf-test"
  image = docker_image.foo_mounts.latest

  mounts {
    target    = "/mount/test"
    source    = docker_volume.foo_mounts.name
    type      = "volume"
    read_only = true
  }
  mounts {
    target = "/mount/tmpfs"
    type   = "tmpfs"
  }
}
