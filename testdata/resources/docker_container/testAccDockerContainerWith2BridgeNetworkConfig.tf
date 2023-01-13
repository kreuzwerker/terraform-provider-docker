resource "docker_network" "tftest" {
  name = "tftest-contnw"
}

resource "docker_network" "tftest_2" {
  name = "tftest-contnw-2"
}

resource "docker_image" "foo" {
  name = "nginx:latest"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id
  networks_advanced {
    name = docker_network.tftest.name
  }
  networks_advanced {
    name = docker_network.tftest_2.name
  }
  # networks = [
  #   docker_network.tftest.name,
  #   docker_network.tftest_2.name
  # ]
}
