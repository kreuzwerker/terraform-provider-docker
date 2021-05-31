resource "docker_plugin" "test" {
  name          = "vieux/sshfs"
  force_destroy = true
  grant_permissions {
    name = "network"
    value = [
      "host"
    ]
  }
  grant_permissions {
    name = "mount"
    value = [
      "",
      "/var/lib/docker/plugins/"
    ]
  }
  grant_permissions {
    name = "device"
    value = [
      "/dev/fuse"
    ]
  }
  grant_permissions {
    name = "capabilities"
    value = [
      "CAP_SYS_ADMIN"
    ]
  }
}
