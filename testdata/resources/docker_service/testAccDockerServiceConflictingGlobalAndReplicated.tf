resource "docker_service" "foo" {
  name = "tftest-service-basic"
  task_spec {
    container_spec {
      image = "127.0.0.1:15000/tftest-service:v1"
    }
  }
  mode {
    replicated {
      replicas = 2
    }
    global = true
  }
}
