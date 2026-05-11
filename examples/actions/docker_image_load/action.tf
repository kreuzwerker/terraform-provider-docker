resource "terraform_data" "load_trigger" {
  triggers_replace = [
    filesha512("./busybox-image.tar")
  ]

  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.docker_image_load.load]
    }
  }
}

action "docker_image_load" "load" {
  config {
    source   = pathexpand("./busybox-image.tar")
    quiet    = true
    platform = "linux/amd64"
  }
}
