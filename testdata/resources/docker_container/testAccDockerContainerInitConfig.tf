resource "docker_image" "fooinit" {
  name = "busybox:1.35.0"
}

resource "docker_container" "fooinit" {
  name  = "tf-test"
  image = docker_image.fooinit.latest
  init  = true
}
