# uses the 'latest' tag
data "docker_image" "latest" {
  name = "nginx"
}

# uses a specific tag
data "docker_image" "specific" {
  name = "nginx:1.17.6"
}

# use the image digest
data "docker_image" "digest" {
  name = "nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2"
}

# uses the tag and the image digest
data "docker_image" "tag_and_digest" {
  name = "nginx:1.19.1@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2"
}
