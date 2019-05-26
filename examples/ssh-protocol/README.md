## Using and testing the ssh-protocol

The `ssh://` which was introduced in docker can be tested/used as shown in this example.

```sh
# export your pub key(s) in terraform pub_key variable
export TF_VAR_pub_key="$(cat ~/.ssh/*.pub)"

# launch dind container with ssh and docker accepting your PK for root user
terraform apply -target docker_container.dind

# wait for few seconds/minutes

# ssh to container to remember server keys
ssh root@localhost -p 32822 uptime

# test docker host ssh protocol
terraform apply -target docker_image.test
```