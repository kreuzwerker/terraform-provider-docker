resource "docker_registry_image" "helloworld" {
  name          = docker_image.image.name
  keep_remotely = true
}

resource "docker_image" "image" {
  name = "registry.com/somename:1.0"
  build {
    context = "${path.cwd}/absolutePathToContextFolder"
  }
}
