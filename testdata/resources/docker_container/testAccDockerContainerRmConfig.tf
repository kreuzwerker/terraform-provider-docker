resource "docker_image" "foo" {
  name         = "busybox:latest"
  keep_locally = true
}
resource "docker_container" "foo" {
  name    = "tf-test"
  image   = docker_image.foo.latest
  command = ["/bin/sleep", "15"]
  rm      = true
}
