resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id

  upload {
    content_base64 = base64encode("894fc3f56edf2d3a4c5fb5cb71df910f958a2ed8")
    file           = "/terraform/test1.txt"
    executable     = true
  }

  upload {
    content = "foobar"
    file    = "/terraform/test2.txt"
  }
}
