resource "docker_image" "foo" {
  name = "busybox:1.34.0"
  platform = "linux/amd64"
}
