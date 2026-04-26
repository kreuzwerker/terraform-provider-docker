## The following code performs an `docker image import` whenever the `import.tar` file changes

resource "terraform_data" "bootstrap" {
  triggers_replace = [
    filesha512("./import.tar")
  ]

  lifecycle {
    action_trigger {
      events  = [after_update]
      actions = [action.docker_image_import.import_export]
    }
  }
}


action "docker_image_import" "import_export" {
  config {
    source    = pathexpand("./import.tar")
    reference = "example-imported-image:latest"
    message   = "imported from a tar archive"
    changes   = ["CMD [\"sh\"]"]
    platform  = "linux/amd64"
  }
}