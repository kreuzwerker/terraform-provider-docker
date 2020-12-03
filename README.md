Terraform Provider
==================

- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)
- Mailing list: [Google Groups](http://groups.google.com/group/terraform-tool)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) 0.12.x
-	[Go](https://golang.org/doc/install) 1.15.x (to build the provider plugin)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/terraform-providers/terraform-provider-docker`

```sh
$ mkdir -p $GOPATH/src/github.com/terraform-providers; cd $GOPATH/src/github.com/terraform-providers
$ git clone git@github.com:terraform-providers/terraform-provider-docker
```

Enter the provider directory and build the provider

```sh
$ cd $GOPATH/src/github.com/terraform-providers/terraform-provider-docker
$ make build
```

Using the provider
----------------------
## Fill in for each provider

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need the latest version of [Go](http://www.golang.org) installed on your machine (currently 1.15). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH` (note that we typically test older versions of golang as long as they are supported upstream, though we recommend new development to happen on the latest release).

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-docker
...
```

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create a local registry which will be deleted afterwards.

```sh
$ make testacc
```

In order to run only single Acceptance tests, execute the following steps:

```sh
# setup the testing environment
$ make testacc_setup

# run single tests
TF_LOG=INFO TF_ACC=1 go test -v ./docker -run ^TestAccDockerImage_data_private_config_file$ -timeout 360s

# cleanup the local testing resources
$ make testacc_cleanup
```

In order to extend the provider and test it with `terraform`, build the provider as mentioned above with
```sh
$ make build
# or a specific version
$ go build -o terraform-provider-docker_v1.2.0_x4
```

Remove an explicit version of the provider you develop, because `terraform` will fetch
the locally built one in `$GOPATH/bin`
```hcl
provider "docker" {
  # version = "~> 0.1.2"
  ...
}
```


Don't forget to run `terraform init` each time you rebuild the provider. Check [here](https://www.youtube.com/watch?v=TMmovxyo5sY&t=30m14s) for a more detailed explanation.

You can check the latest released version of a provider at https://releases.hashicorp.com/terraform-provider-docker/.

Developing on Windows
---------------------

You can build and test on Widows without `make`.  Run `go install` to
build and `Scripts\runAccTests.bat` to run the test suite.

Continuous integration for Windows is not available at the moment due
to lack of a CI provider that is free for open source projects *and*
supports running Linux containers in Docker for Windows.  For example,
AppVeyor is free for open source projects and provides Docker on its
Windows builds, but only offers Linux containers on Windows as a paid
upgrade.
