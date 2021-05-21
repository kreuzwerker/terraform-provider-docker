provider "docker" {
  host = "tcp://your-host-ip:2376/"

  # -> specify either
  cert_path = pathexpand("~/.docker")

  # -> or the following
  ca_material   = file(pathexpand("~/.docker/ca.pem")) # this can be omitted
  cert_material = file(pathexpand("~/.docker/cert.pem"))
  key_material  = file(pathexpand("~/.docker/key.pem"))
}
