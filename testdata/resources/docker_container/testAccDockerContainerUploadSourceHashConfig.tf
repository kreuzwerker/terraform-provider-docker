resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id

  upload {
    source      = "%s"
    source_hash = "%s"
    file        = "/terraform/test.txt"
    executable  = true
  }
}
