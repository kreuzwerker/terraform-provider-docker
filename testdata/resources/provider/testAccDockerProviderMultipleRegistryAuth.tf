provider "docker" {
    alias = "private"
    registry_auth {
      address  = "%s"
    }
    registry_auth {
      address  = "public.ecr.aws"
      username = "test"
      password = "user"
    }
}
data "docker_registry_image" "foobar" {
    provider             = "docker.private"
    name                 = "127.0.0.1:15000/tftest-service:v1"
    insecure_skip_verify = true
}