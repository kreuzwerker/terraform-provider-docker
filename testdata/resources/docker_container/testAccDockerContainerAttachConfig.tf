resource "docker_image" "foo" {
  name         = "busybox:latest"
  keep_locally = true
}
resource "docker_container" "foo" {
  name     = "tf-test"
  image    = docker_image.foo.image_id
  command  = ["/bin/sh", "-c", "for i in $(seq 1 15); do sleep 1; done"]
  attach   = true
  must_run = false
}
