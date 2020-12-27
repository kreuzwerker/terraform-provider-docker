# Contributing

By participating to this project, you agree to abide our [code of
conduct](/CODE_OF_CONDUCT.md).

## Setup your machine

`terraform-provider-docker` is written in [Go](https://golang.org/).

Prerequisites:

- `make`, `git`, `bash`
- [Go 1.15+](https://golang.org/doc/install)
- [Docker](https://www.docker.com/)
- [Terraform 0.12+](https://terraform.io/)

Clone `terraform-provider-docker` anywhere:

```sh
git clone git@github.com:kreuzwerker/terraform-provider-docker.git
```

Install the build dependencies:

```sh
make build
```

## Test your change

You can create a branch for your changes and try to build from the source as you go:

```sh
make build
```

### Unit and acceptance tests
When you are satisfied with the changes, **tests**, and **documentation** updates, we suggest you run:

```sh
# unit tests
make test

# acceptance test
## setup the testing environment
make testacc_setup

## run a single test
TF_LOG=INFO TF_ACC=1 go test -v ./docker -run ^TestAccDockerImage_data_private_config_file$ -timeout 360s

## cleanup the local testing resources
make testacc_cleanup
```

Furthermore, we recommened running the linters for the code and the documentation:

```sh
# install all the dependencies
make setup
make golangci-lint
make website-link-check
make website-lint
# you can also use this command to fix most errors automatically
make website-lint-fix
```

### Test against current terraform IaC descriptions
In order to extend the provider and test it with `terraform`, build the provider as mentioned above with:

```sh
# Testing in a local mirror which needs to have the following convention.
# See https://www.terraform.io/docs/commands/cli-config.html#provider-installation for details
export TESTING_MIRROR=testing-mirror/registry.terraform.io/kreuzwerker/docker/9.9.9/$(go env GOHOSTOS)_$(go env GOHOSTARCH)
mkdir -p ./$TESTING_MIRROR

# now we build into the provider into the local mirror
go build -o ./$TESTING_MIRROR/terraform-provider-docker_v9.9.9
```

Now we change into the `testing` directory (which is ignored as well) and set an explicit version of the provider we develop:
```hcl
terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "9.9.9"
    }
  }
}

provider "docker" {
}

resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name    = "foo"
  image   = docker_image.foo.latest
}
```

As the next step we can init terraform by provider a local plugin directory:
```sh
# Which reflects the convention mentioned before
# See https://www.terraform.io/docs/commands/init.html#plugin-installation
terraform init -plugin-dir=../testing-mirror
terraform plan
terraform apply -auto-approve
```

### Developing on Windows

You can build and test on Windows without `make`.  Run `go install` to
build and `Scripts\runAccTests.bat` to run the test suite.

Continuous integration for Windows is not available at the moment due
to lack of a CI provider that is free for open source projects *and*
supports running Linux containers in Docker for Windows.  For example,
AppVeyor is free for open source projects and provides Docker on its
Windows builds, but only offers Linux containers on Windows as a paid
upgrade.

## Create a commit

Commit messages should be well formatted, and to make that "standardized", we
are using Conventional Commits.

You can follow the documentation on
[their website](https://www.conventionalcommits.org).

## Submit a pull request

Push your branch to your `terraform-provider-docker` fork and open a 
pull request against the master branch.