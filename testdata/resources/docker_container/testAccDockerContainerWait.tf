resource "docker_image" "this" {
  name = "busybox:latest"
  keep_locally = true
}

resource "docker_container" "this" {
  name  = "tf-test"
  image = docker_image.this.image_id
  wait  = true
  rm = false
}
