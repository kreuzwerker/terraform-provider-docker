provider "docker" {
  host = "tcp://127.0.0.1:1"
}

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
    host = "%[1]s"
  }
}

data "docker_network" "bridge" {
  name = "bridge"

  daemon {
    host = "%[1]s"
  }
}
