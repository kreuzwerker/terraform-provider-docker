resource "docker_image" "foo" {
  name                 = "localhost:15000/foo:1.0"
  build {
    suppress_output = true
    remote_context  = "fooRemoteContext"
    no_cache        = true
    remove          = true
    force_remove    = true
    pull_parent     = true
    isolation       = "hyperv"
    cpu_set_cpus    = "fooCpuSetCpus"
    cpu_set_mems    = "fooCpuSetMems"
    cpu_shares      = 4
    cpu_quota       = 5
    cpu_period      = 6
    memory          = 1
    memory_swap     = 2
    cgroup_parent   = "fooCgroupParent"
    network_mode    = "fooNetworkMode"
    shm_size        = 3
    dockerfile      = "fooDockerfile"
    ulimit {
      name = "foo"
      hard = 1
      soft = 2
    }
    auth_config {
      host_name      = "foo.host"
      user_name      = "fooUserName"
      password       = "fooPassword"
      auth           = "fooAuth"
      email          = "fooEmail"
      server_address = "fooServerAddress"
      identity_token = "fooIdentityToken"
      registry_token = "fooRegistryToken"

    }
    build_args = {
      "HTTP_PROXY" = "http://10.20.30.2:1234"
    }
    context = "context"
    labels = {
      foo = "bar"
    }
    squash       = true
    cache_from   = ["fooCacheFrom", "barCacheFrom"]
    security_opt = ["fooSecurityOpt", "barSecurityOpt"]
    extra_hosts  = ["fooExtraHost", "barExtraHost"]
    target       = "fooTarget"
    session_id   = "fooSessionId"
    platform     = "fooPlatform"
    version      = "1"
    build_id     = "fooBuildId"
  }
}
