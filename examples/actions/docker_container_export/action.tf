resource "docker_image" "busybox" {
  name = "busybox:1.35.0"
}

resource "docker_container" "target" {
  name     = "docker-container-export-example"
  image    = docker_image.busybox.image_id
  must_run = true
  command  = ["sh", "-c", "sleep 300"]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_container_export.export]
    }
  }
}

action "docker_container_export" "export" {
  config {
    container = docker_container.target.name
    output    = pathexpand("./container-export.tar")
  }
}
