resource "docker_network" "tftest" {
  name = "tftest-contnw"
}

resource "docker_network" "tftest_2" {
  name = "tftest-contnw-2"
}

resource "docker_image" "foo" {
  name = "busybox:1.35.0"
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest
  networks = [
    docker_network.tftest.name,
    docker_network.tftest_2.name
  ]
}
