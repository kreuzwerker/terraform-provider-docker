resource "docker_network" "test_1" {
  name = "tf-test-mac-1"
}

resource "docker_network" "test_2" {
  name = "tf-test-mac-2"
}

resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id
  networks_advanced {
    name        = docker_network.test_1.name
    mac_address = "02:42:ac:11:00:aa"
  }
  networks_advanced {
    name        = docker_network.test_2.name
    mac_address = "02:42:ac:11:00:bb"
  }
}
