resource "docker_image" "foo" {
  name = "nginx:latest"

  daemon {
    host = "%[1]s"
  }
}

resource "docker_container" "foo" {
  name  = "tf-test-daemon"
  image = docker_image.foo.image_id

  daemon {
    host     = "%[1]s"
    ssh_opts = ["-o", "ConnectTimeout=5"]
  }
}
