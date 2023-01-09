resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_network" "test_network_1" {
  name = "tftest-1"
}

resource "docker_network" "test_network_2" {
  name = "tftest-2"
}

resource "docker_container" "foo" {
  name          = "tf-test"
  image         = docker_image.foo.image_id
  network_mode  = docker_network.test_network_1.name
  networks_advanced {
    name = docker_network.test_network_2.name
    aliases = ["tftest-container"]
  }
}

resource "docker_container" "bar" {
  name          = "tf-test-bar"
  image         = docker_image.foo.image_id
  network_mode  = "bridge"
  networks_advanced {
    name = docker_network.test_network_2.name
    aliases = ["tftest-container-foo"]
  }
}
