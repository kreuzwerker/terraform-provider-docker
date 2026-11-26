provider "docker" {
  host = "tcp://127.0.0.1:1"
}

resource "terraform_data" "vm" {
  input = "%[1]s"
}

resource "docker_image" "foo" {
  name = "nginx:latest"

  daemon {
    host = terraform_data.vm.output
  }
}

resource "docker_container" "foo" {
  name  = "tf-test-daemon"
  image = docker_image.foo.image_id

  daemon {
    host = terraform_data.vm.output
  }
}
