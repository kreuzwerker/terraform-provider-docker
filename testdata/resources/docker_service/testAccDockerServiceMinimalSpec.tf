provider "docker" {
  registry_auth {
    address = "127.0.0.1:15000"
  }
}

resource "docker_service" "foo" {
  name = "tftest-service-basic"
  task_spec {
    container_spec {
      image             = "127.0.0.1:15000/tftest-service:v1@sha256:2ca4c7a50df3515ea96106caab374759879830f6e4d6b400cee064e2e8db08c0"
      stop_grace_period = "10s"
    }
  }
}
