resource "docker_registry_image" "helloworld" {
  name = "helloworld:1.0"

  build {
    context = "pathToContextFolder"
  }
}