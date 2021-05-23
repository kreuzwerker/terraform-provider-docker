resource "docker_secret" "foo" {
  name = "foo"
  data = base64encode("{\"foo\": \"s3cr3t\"}")
}
