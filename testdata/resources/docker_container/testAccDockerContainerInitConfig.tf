resource "docker_image" "fooinit" {
  name = "nginx:latest"
}

resource "docker_container" "fooinit" {
  name  = "tf-test"
  image = docker_image.fooinit.image_id
  init  = true
}
