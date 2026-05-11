resource "docker_image" "busybox" {
  name = "busybox:1.35.0"
}

resource "terraform_data" "save_trigger" {
  depends_on = [docker_image.busybox]

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.docker_image_save.save]
    }
  }
}

action "docker_image_save" "save" {
  config {
    images   = [docker_image.busybox.name]
    output   = pathexpand("./busybox-image.tar")
    platform = "linux/amd64"
  }
}
