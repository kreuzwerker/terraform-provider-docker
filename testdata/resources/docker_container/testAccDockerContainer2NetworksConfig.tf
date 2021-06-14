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
  image         = docker_image.foo.latest
  network_mode  = docker_network.test_network_1.name
  networks      = [docker_network.test_network_2.name]
  network_alias = ["tftest-container"]
}

resource "docker_container" "bar" {
  name          = "tf-test-bar"
  image         = docker_image.foo.latest
  network_mode  = "bridge"
  networks      = [docker_network.test_network_2.name]
  network_alias = ["tftest-container-foo"]
}
