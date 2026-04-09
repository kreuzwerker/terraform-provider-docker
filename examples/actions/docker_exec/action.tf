resource "docker_image" "busybox" {
  name = "busybox:1.35.0"
}

resource "docker_container" "target" {
  name     = "docker-exec-example"
  image    = docker_image.busybox.image_id
  must_run = true
  command  = ["sh", "-c", "sleep 300"]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_exec.create_file]
    }
  }
}

action "docker_exec" "create_file" {
  config {
    container = docker_container.target.name
    command   = ["sh", "-c", "touch /tmp/created-by-action"]
  }
}
