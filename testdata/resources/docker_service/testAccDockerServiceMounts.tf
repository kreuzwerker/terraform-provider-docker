provider "docker" {
  registry_auth {
    address = "127.0.0.1:15000"
  }
}

resource "docker_volume" "test_volume" {
  name = "tftest-volume"
}

resource "docker_service" "foo_empty" {
  name = "tftest-service-mount-bind-empty"
  task_spec {
    container_spec {
      image             = "127.0.0.1:15000/tftest-service:v1@sha256:2ca4c7a50df3515ea96106caab374759879830f6e4d6b400cee064e2e8db08c0"
      stop_grace_period = "10s"

      mounts {
        target    = "/mount/test"
        source    = docker_volume.test_volume.name
        type      = "bind"
        read_only = true

        bind_options {}
      }
    }
  }
}
resource "docker_service" "foo_null" {
  name = "tftest-service-mount-bind-null"
  task_spec {
    container_spec {
      image             = "127.0.0.1:15000/tftest-service:v1@sha256:2ca4c7a50df3515ea96106caab374759879830f6e4d6b400cee064e2e8db08c0"
      stop_grace_period = "10s"

      mounts {
        target    = "/mount/test"
        source    = docker_volume.test_volume.name
        type      = "bind"
        read_only = true

        bind_options {
          propagation = null
        }
      }
    }
  }
}
