#!/bin/bash

# Import an existing Docker Buildx builder
terraform import docker_buildx_builder.example my-existing-builder

# Note: After import, you may want to enable auto_recreate for the imported builder
# to ensure it gets recreated if missing from Docker in the future
