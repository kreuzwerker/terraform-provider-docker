# Changelog

<a name="v4.1.0"></a>
## [v4.1.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v4.0.0...v4.1.0) (2026-04-09)



### Chore

* Prepare release 4.1.0 (#896) ([#896](https://github.com/kreuzwerker/terraform-provider-docker/issues/896))


### Feat

* Implement docker_exec action (#885) ([#885](https://github.com/kreuzwerker/terraform-provider-docker/issues/885))

* Allow `docker_registry_image.auth_config` to mirror provider `registry_auth` optional credentials (#887) ([#887](https://github.com/kreuzwerker/terraform-provider-docker/issues/887))

* Add `platform` support to `docker_container` for cross-architecture emulation (#886) ([#886](https://github.com/kreuzwerker/terraform-provider-docker/issues/886))

* Add Plugin Framework `docker_containers` data source for Docker container enumeration (#893) ([#893](https://github.com/kreuzwerker/terraform-provider-docker/issues/893))


### Fix

* update module github.com/containerd/platforms to v1.0.0-rc.4 (#889) ([#889](https://github.com/kreuzwerker/terraform-provider-docker/issues/889))

* Make container deletion idempotent for missing containers (#891) ([#891](https://github.com/kreuzwerker/terraform-provider-docker/issues/891) [#890](https://github.com/kreuzwerker/terraform-provider-docker/issues/890))

* Fix `docker_service` platform flattening hash to prevent platform list drift on updates (#892) ([#892](https://github.com/kreuzwerker/terraform-provider-docker/issues/892))


### Other

* Avoid `docker_container` replacement when only daemon default `log_opts` are present (#888) ([#888](https://github.com/kreuzwerker/terraform-provider-docker/issues/888))

* Prevent `docker_container` read panic with CDI `device_requests` by hardening device flattening (#895) ([#895](https://github.com/kreuzwerker/terraform-provider-docker/issues/895))


<a name="v4.0.0"></a>
## [v4.0.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.9.0...v4.0.0) (2026-04-03)



### Chore

* Add deprecation for docker_service.networks_advanced.name (#837) ([#837](https://github.com/kreuzwerker/terraform-provider-docker/issues/837))

* update actions/checkout action to v6 (#825) ([#825](https://github.com/kreuzwerker/terraform-provider-docker/issues/825))

* update hashicorp/setup-terraform action to v4 (#860) ([#860](https://github.com/kreuzwerker/terraform-provider-docker/issues/860))

* update docker/setup-docker-action action to v5 (#866) ([#866](https://github.com/kreuzwerker/terraform-provider-docker/issues/866))

* update dependency golangci/golangci-lint to v2.10.1 (#869) ([#869](https://github.com/kreuzwerker/terraform-provider-docker/issues/869))

* update dependency golangci/golangci-lint to v2.11.4 (#871) ([#871](https://github.com/kreuzwerker/terraform-provider-docker/issues/871))

* Prepare 4.0.0 release (#884) ([#884](https://github.com/kreuzwerker/terraform-provider-docker/issues/884))


### Feat

* Add muxing to introduce new plugin framework (#838) ([#838](https://github.com/kreuzwerker/terraform-provider-docker/issues/838))

* Multiple enhancements (#854) ([#854](https://github.com/kreuzwerker/terraform-provider-docker/issues/854) [#543](https://github.com/kreuzwerker/terraform-provider-docker/issues/543) [#777](https://github.com/kreuzwerker/terraform-provider-docker/issues/777) [#588](https://github.com/kreuzwerker/terraform-provider-docker/issues/588))

* Make buildx builder default (#855) ([#855](https://github.com/kreuzwerker/terraform-provider-docker/issues/855))

* Add new docker container attributes (#857) ([#857](https://github.com/kreuzwerker/terraform-provider-docker/issues/857))

* Add CDI device support (#762) ([#762](https://github.com/kreuzwerker/terraform-provider-docker/issues/762))

* Implement proper parsing of GPU device requests when using gpus… (#881) ([#881](https://github.com/kreuzwerker/terraform-provider-docker/issues/881))

* add selinux_relabel attribute to docker_container volumes (#883) ([#883](https://github.com/kreuzwerker/terraform-provider-docker/issues/883))


### Fix

* update module golang.org/x/sync to v0.19.0 (#828) ([#828](https://github.com/kreuzwerker/terraform-provider-docker/issues/828))

* update module github.com/hashicorp/terraform-plugin-log to v0.10.0 (#823) ([#823](https://github.com/kreuzwerker/terraform-provider-docker/issues/823))

* update module github.com/morikuni/aec to v1.1.0 (#829) ([#829](https://github.com/kreuzwerker/terraform-provider-docker/issues/829))

* update module google.golang.org/protobuf to v1.36.11 (#830) ([#830](https://github.com/kreuzwerker/terraform-provider-docker/issues/830))

* update module github.com/sirupsen/logrus to v1.9.4 (#836) ([#836](https://github.com/kreuzwerker/terraform-provider-docker/issues/836))

* Refactor docker container state handling to properly restart when exited (#841) ([#841](https://github.com/kreuzwerker/terraform-provider-docker/issues/841))

* docker container stopped ports (#842) ([#842](https://github.com/kreuzwerker/terraform-provider-docker/issues/842))

* correctly set docker_container devices (#843) ([#843](https://github.com/kreuzwerker/terraform-provider-docker/issues/843))

* update module github.com/katbyte/terrafmt to v0.5.6 (#844) ([#844](https://github.com/kreuzwerker/terraform-provider-docker/issues/844))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.38.2 (#847) ([#847](https://github.com/kreuzwerker/terraform-provider-docker/issues/847))

* Use DOCKER_CONFIG env same way as with docker cli (#849) ([#849](https://github.com/kreuzwerker/terraform-provider-docker/issues/849))

* calculation of Dockerfile path in docker_image build (#853) ([#853](https://github.com/kreuzwerker/terraform-provider-docker/issues/853))

* update module github.com/hashicorp/terraform-plugin-go to v0.30.0 (#861) ([#861](https://github.com/kreuzwerker/terraform-provider-docker/issues/861))

* update module github.com/hashicorp/terraform-plugin-framework to v1.18.0 (#862) ([#862](https://github.com/kreuzwerker/terraform-provider-docker/issues/862))

* update module github.com/hashicorp/terraform-plugin-mux to v0.22.0 (#863) ([#863](https://github.com/kreuzwerker/terraform-provider-docker/issues/863))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.39.0 (#864) ([#864](https://github.com/kreuzwerker/terraform-provider-docker/issues/864))

* update module golang.org/x/sync to v0.20.0 (#872) ([#872](https://github.com/kreuzwerker/terraform-provider-docker/issues/872))

* Handle size_bytes in tmpfs_options in docker_service (#882) ([#882](https://github.com/kreuzwerker/terraform-provider-docker/issues/882))

* tests for healthcheck is not required for docker container resource (#834) ([#834](https://github.com/kreuzwerker/terraform-provider-docker/issues/834))


### Other

* Prevent `docker_registry_image` panic on registries returning nil body without digest header (#880) ([#880](https://github.com/kreuzwerker/terraform-provider-docker/issues/880))


<a name="v3.9.0"></a>
## [v3.9.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.8.0...v3.9.0) (2025-11-09)



### Chore

* Prepare release v3.8.0 (#806) ([#806](https://github.com/kreuzwerker/terraform-provider-docker/issues/806))

* Add file requested by hashicorp (#813) ([#813](https://github.com/kreuzwerker/terraform-provider-docker/issues/813))

* update golangci/golangci-lint-action action to v9 (#819) ([#819](https://github.com/kreuzwerker/terraform-provider-docker/issues/819))

* Prepare release v3.9.0 (#821) ([#821](https://github.com/kreuzwerker/terraform-provider-docker/issues/821))


### Feat

* Implement caching of docker provider (#808) ([#808](https://github.com/kreuzwerker/terraform-provider-docker/issues/808))


### Fix

* update module github.com/hashicorp/terraform-plugin-docs to v0.24.0 (#807) ([#807](https://github.com/kreuzwerker/terraform-provider-docker/issues/807))

* docker_service label can be updated without recreate (#814) ([#814](https://github.com/kreuzwerker/terraform-provider-docker/issues/814))

* test attribute of docker_service healthcheck is not required (#815) ([#815](https://github.com/kreuzwerker/terraform-provider-docker/issues/815))

* update module github.com/docker/cli to v28.5.2+incompatible (#816) ([#816](https://github.com/kreuzwerker/terraform-provider-docker/issues/816))

* update module github.com/docker/docker to v28.5.2+incompatible (#817) ([#817](https://github.com/kreuzwerker/terraform-provider-docker/issues/817))

* update module golang.org/x/sync to v0.18.0 (#820) ([#820](https://github.com/kreuzwerker/terraform-provider-docker/issues/820))


<a name="v3.8.0"></a>
## [v3.8.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.7.0...v3.8.0) (2025-10-08)



### Feat

* Implement docker cluster volume (#793) ([#793](https://github.com/kreuzwerker/terraform-provider-docker/issues/793))

* implement mac_address for networks_advanced (#794) ([#794](https://github.com/kreuzwerker/terraform-provider-docker/issues/794))

* Add build option for additional contexts (#798) ([#798](https://github.com/kreuzwerker/terraform-provider-docker/issues/798))

* Add build attribute for docker_registry_image (#805) ([#805](https://github.com/kreuzwerker/terraform-provider-docker/issues/805))


### Fix

* update module google.golang.org/protobuf to v1.36.8 (#775) ([#775](https://github.com/kreuzwerker/terraform-provider-docker/issues/775))

* update module google.golang.org/protobuf to v1.36.9 (#787) ([#787](https://github.com/kreuzwerker/terraform-provider-docker/issues/787))

* omit sending systempaths=unconfied to daemon (#796) ([#796](https://github.com/kreuzwerker/terraform-provider-docker/issues/796))

* Recreate builder if deleted out-of-band (#797) ([#797](https://github.com/kreuzwerker/terraform-provider-docker/issues/797))

* update module golang.org/x/sync to v0.17.0 (#785) ([#785](https://github.com/kreuzwerker/terraform-provider-docker/issues/785))

* update module github.com/hashicorp/terraform-plugin-docs to v0.23.0 (#788) ([#788](https://github.com/kreuzwerker/terraform-provider-docker/issues/788))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.38.1 (#789) ([#789](https://github.com/kreuzwerker/terraform-provider-docker/issues/789))

* update module github.com/docker/cli to v28.4.0+incompatible (#781) ([#781](https://github.com/kreuzwerker/terraform-provider-docker/issues/781))

* update module google.golang.org/protobuf to v1.36.10 (#799) ([#799](https://github.com/kreuzwerker/terraform-provider-docker/issues/799))

* update module github.com/docker/cli to v28.5.0+incompatible (#800) ([#800](https://github.com/kreuzwerker/terraform-provider-docker/issues/800))

* update module github.com/docker/docker to v28.5.0+incompatible (#782) ([#782](https://github.com/kreuzwerker/terraform-provider-docker/issues/782))

* update module github.com/docker/docker to v28.5.1+incompatible (#804) ([#804](https://github.com/kreuzwerker/terraform-provider-docker/issues/804))

* update module github.com/docker/cli to v28.5.1+incompatible (#803) ([#803](https://github.com/kreuzwerker/terraform-provider-docker/issues/803))


<a name="v3.7.0"></a>
## [v3.7.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.6.2...v3.7.0) (2025-08-19)



### Chore

* update dependency go to v1.24.5 (#757) ([#757](https://github.com/kreuzwerker/terraform-provider-docker/issues/757))

* update dependency go to v1.24.6 (#764) ([#764](https://github.com/kreuzwerker/terraform-provider-docker/issues/764))

* update actions/checkout action to v5 (#768) ([#768](https://github.com/kreuzwerker/terraform-provider-docker/issues/768))

* Prepare release v3.7.0 (#774) ([#774](https://github.com/kreuzwerker/terraform-provider-docker/issues/774))


### Feat

* Add timeout support to docker_registry_image resource (#769) ([#769](https://github.com/kreuzwerker/terraform-provider-docker/issues/769))

* Implement cache_from and cache_to for docker_image (#772) ([#772](https://github.com/kreuzwerker/terraform-provider-docker/issues/772))

* Implement memory_reservation and network_mode enhancements (#773) ([#773](https://github.com/kreuzwerker/terraform-provider-docker/issues/773))


### Fix

* update module github.com/hashicorp/terraform-plugin-docs to v0.22.0 (#755) ([#755](https://github.com/kreuzwerker/terraform-provider-docker/issues/755))

* update module golang.org/x/sync to v0.16.0 (#758) ([#758](https://github.com/kreuzwerker/terraform-provider-docker/issues/758))

* update module google.golang.org/protobuf to v1.36.7 (#765) ([#765](https://github.com/kreuzwerker/terraform-provider-docker/issues/765))

* update module github.com/docker/go-connections to v0.6.0 (#766) ([#766](https://github.com/kreuzwerker/terraform-provider-docker/issues/766))

* Correctly get and set nanoCPUs for docker_container (#771) ([#771](https://github.com/kreuzwerker/terraform-provider-docker/issues/771))


<a name="v3.6.2"></a>
## [v3.6.2](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.6.1...v3.6.2) (2025-06-13)



### Chore

* Prepare release v3.6.2 (#750) ([#750](https://github.com/kreuzwerker/terraform-provider-docker/issues/750))


### Feat

* Allow digest in image name (#744) ([#744](https://github.com/kreuzwerker/terraform-provider-docker/issues/744))


### Fix

* Typo in cgroup_parent handling (#746) ([#746](https://github.com/kreuzwerker/terraform-provider-docker/issues/746) [#745](https://github.com/kreuzwerker/terraform-provider-docker/issues/745))

* Reading non existant volume should recreate (#749) ([#749](https://github.com/kreuzwerker/terraform-provider-docker/issues/749))

* Remove wrong buildkit version assignment (#747) ([#747](https://github.com/kreuzwerker/terraform-provider-docker/issues/747))


<a name="v3.6.1"></a>
## [v3.6.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.6.0...v3.6.1) (2025-06-05)



### Chore

* Add retryon429 for markdownlint (#736) ([#736](https://github.com/kreuzwerker/terraform-provider-docker/issues/736))

* update dependency go to v1.24.4 (#742) ([#742](https://github.com/kreuzwerker/terraform-provider-docker/issues/742))

* Prepare release v3.6.1 (#743) ([#743](https://github.com/kreuzwerker/terraform-provider-docker/issues/743))


### Feat

* allow to set the cgroup parent for container (#609) ([#609](https://github.com/kreuzwerker/terraform-provider-docker/issues/609))


### Fix

* update module golang.org/x/sync to v0.15.0 (#741) ([#741](https://github.com/kreuzwerker/terraform-provider-docker/issues/741))


<a name="v3.6.0"></a>
## [v3.6.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.5.0...v3.6.0) (2025-05-25)



### Chore

* update dependency go to v1.24.3 (#719) ([#719](https://github.com/kreuzwerker/terraform-provider-docker/issues/719))

* update golangci/golangci-lint-action action to v8 (#718) ([#718](https://github.com/kreuzwerker/terraform-provider-docker/issues/718))

* Prepare release v3.6.0 (#735) ([#735](https://github.com/kreuzwerker/terraform-provider-docker/issues/735))


### Feat

* implement Buildx builder resource (#724) ([#724](https://github.com/kreuzwerker/terraform-provider-docker/issues/724))

* Add implementaion of capabilities in docker servic (#727) ([#727](https://github.com/kreuzwerker/terraform-provider-docker/issues/727))

* Implement correct cpu scheduler settings (#732) ([#732](https://github.com/kreuzwerker/terraform-provider-docker/issues/732))


### Fix

* update module google.golang.org/protobuf to v1.36.6 (#720) ([#720](https://github.com/kreuzwerker/terraform-provider-docker/issues/720))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.37.0 (#726) ([#726](https://github.com/kreuzwerker/terraform-provider-docker/issues/726))

* update module github.com/containerd/console to v1.0.5 (#729) ([#729](https://github.com/kreuzwerker/terraform-provider-docker/issues/729))

* Make endpoint validation less strict (#733) ([#733](https://github.com/kreuzwerker/terraform-provider-docker/issues/733))

* update module github.com/moby/buildkit to v0.22.0 (#731) ([#731](https://github.com/kreuzwerker/terraform-provider-docker/issues/731))

* Implement buildx fixes for general buildkit support and platform handling (#734) ([#734](https://github.com/kreuzwerker/terraform-provider-docker/issues/734))


<a name="v3.5.0"></a>
## [v3.5.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.4.0...v3.5.0) (2025-05-06)



### Chore

* Prepare release v3.5.0 (#721) ([#721](https://github.com/kreuzwerker/terraform-provider-docker/issues/721))


### Feat

* Implement healthcheck start interval (#713) ([#713](https://github.com/kreuzwerker/terraform-provider-docker/issues/713))

* Implement registry_image_manifests data source (#714) ([#714](https://github.com/kreuzwerker/terraform-provider-docker/issues/714))

* Support registries that return empty auth scope #646 ([#646](https://github.com/kreuzwerker/terraform-provider-docker/issues/646))

* Implement using of buildx for docker_image (#717) ([#717](https://github.com/kreuzwerker/terraform-provider-docker/issues/717))


<a name="v3.4.0"></a>
## [v3.4.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.3.0...v3.4.0) (2025-04-25)



### Chore

* Prepare release v3.4.0 (#712) ([#712](https://github.com/kreuzwerker/terraform-provider-docker/issues/712))


### Feat

* Implement volume_options subpath (#710) ([#710](https://github.com/kreuzwerker/terraform-provider-docker/issues/710))


### Fix

* Use auth_config block also for registry_image delete functionality (#708) ([#708](https://github.com/kreuzwerker/terraform-provider-docker/issues/708))

* Improve container wait handling (#709) ([#709](https://github.com/kreuzwerker/terraform-provider-docker/issues/709))

* Prevent recreation of image name is intentionally set to a fixed value (#711) ([#711](https://github.com/kreuzwerker/terraform-provider-docker/issues/711))


<a name="v3.3.0"></a>
## [v3.3.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.2.0...v3.3.0) (2025-04-19)



### Chore

* Update docker/docker and docker/cli to newest stable (#695) ([#695](https://github.com/kreuzwerker/terraform-provider-docker/issues/695))

* Update terraform-plugin-sdk/v2 dependency (#699) ([#699](https://github.com/kreuzwerker/terraform-provider-docker/issues/699))

* update dependency go to v1.24.2 (#700) ([#700](https://github.com/kreuzwerker/terraform-provider-docker/issues/700))

* Prepare release v3.3.0 (#705) ([#705](https://github.com/kreuzwerker/terraform-provider-docker/issues/705))


### Feat

* Implement auth_config for docker_registry_image (#701) ([#701](https://github.com/kreuzwerker/terraform-provider-docker/issues/701))

* Implement tag triggers for docker_tag resource (#702) ([#702](https://github.com/kreuzwerker/terraform-provider-docker/issues/702))

* disable_docker_daemon_check for provider (#703) ([#703](https://github.com/kreuzwerker/terraform-provider-docker/issues/703))

* Implement support for docker context (#704) ([#704](https://github.com/kreuzwerker/terraform-provider-docker/issues/704))


### Fix

* Store correctly ports from server (#698) ([#698](https://github.com/kreuzwerker/terraform-provider-docker/issues/698))


<a name="v3.2.0"></a>
## [v3.2.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.1.2...v3.2.0) (2025-04-16)



### Chore

* Upgrade golangci-lint to next major version (#686) ([#686](https://github.com/kreuzwerker/terraform-provider-docker/issues/686))

* Prepare release v3.2.0 (#694) ([#694](https://github.com/kreuzwerker/terraform-provider-docker/issues/694))


### Docs

* Consolidated update of docs from several PRs (#691) ([#691](https://github.com/kreuzwerker/terraform-provider-docker/issues/691))


### Feat

* Add support for build-secrets (#604) ([#604](https://github.com/kreuzwerker/terraform-provider-docker/issues/604))

* Implement docker_image timeouts (#692) ([#692](https://github.com/kreuzwerker/terraform-provider-docker/issues/692))

* Implement upload permissions in docker_container resource (#693) ([#693](https://github.com/kreuzwerker/terraform-provider-docker/issues/693))


### Fix

* Authentication to ECR public (#690) ([#690](https://github.com/kreuzwerker/terraform-provider-docker/issues/690))


<a name="v3.1.2"></a>
## [v3.1.2](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.1.1...v3.1.2) (2025-04-15)



### Other

* prepare release 3.1.2 (#688) ([#688](https://github.com/kreuzwerker/terraform-provider-docker/issues/688))


<a name="v3.1.1"></a>
## [v3.1.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.1.0...v3.1.1) (2025-04-14)



### Other

* Prepare release 3.1.1 (#687) ([#687](https://github.com/kreuzwerker/terraform-provider-docker/issues/687))


<a name="v3.1.0"></a>
## [v3.1.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.0.2...v3.1.0) (2025-04-14)



### Chore

* update Go version to 1.22 for consistency across workflows, jo… (#613) ([#613](https://github.com/kreuzwerker/terraform-provider-docker/issues/613))

* update golangci/golangci-lint-action action to v6 (#615) ([#615](https://github.com/kreuzwerker/terraform-provider-docker/issues/615))

* update hashicorp/setup-terraform action to v3 (#616) ([#616](https://github.com/kreuzwerker/terraform-provider-docker/issues/616))

* Prepare release 3.1.0 (#685) ([#685](https://github.com/kreuzwerker/terraform-provider-docker/issues/685))


### Feat

* support setting cpu shares (#575) ([#575](https://github.com/kreuzwerker/terraform-provider-docker/issues/575))


### Fix

* update module github.com/katbyte/terrafmt to v0.5.3 (#614) ([#614](https://github.com/kreuzwerker/terraform-provider-docker/issues/614))

* update module github.com/golangci/golangci-lint to v1.59.0 (#474) ([#474](https://github.com/kreuzwerker/terraform-provider-docker/issues/474))

* Set correct default network driver and fix a test (#677) ([#677](https://github.com/kreuzwerker/terraform-provider-docker/issues/677))

* Compress build context before sending it to Docker (#461) ([#461](https://github.com/kreuzwerker/terraform-provider-docker/issues/461) [#439](https://github.com/kreuzwerker/terraform-provider-docker/issues/439))

* update module github.com/katbyte/terrafmt to v0.5.4 (#654) ([#654](https://github.com/kreuzwerker/terraform-provider-docker/issues/654))

* update module github.com/docker/cli to v20.10.27+incompatible (#675) ([#675](https://github.com/kreuzwerker/terraform-provider-docker/issues/675))

* update module github.com/hashicorp/go-cty to v1.5.0 (#679) ([#679](https://github.com/kreuzwerker/terraform-provider-docker/issues/679))

* update module github.com/docker/go-connections to v0.5.0 (#673) ([#673](https://github.com/kreuzwerker/terraform-provider-docker/issues/673))

* Use build_args everywhere and update documentation (#681) ([#681](https://github.com/kreuzwerker/terraform-provider-docker/issues/681))

* update module github.com/docker/docker to v20.10.27+incompatible (#509) ([#509](https://github.com/kreuzwerker/terraform-provider-docker/issues/509))


### Other

* s/presend/present/ (#606) ([#606](https://github.com/kreuzwerker/terraform-provider-docker/issues/606))

* update-dockerignore-for-context-ignore (#625) ([#625](https://github.com/kreuzwerker/terraform-provider-docker/issues/625))

* ✨ Update Terraform version to 1.8.x in acceptance test GitHub Actions workflow. (#626) ([#626](https://github.com/kreuzwerker/terraform-provider-docker/issues/626))

* Ignore error message case in comparison for ignorable messages. (#583) ([#583](https://github.com/kreuzwerker/terraform-provider-docker/issues/583))


<a name="v3.0.2"></a>
## [v3.0.2](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.0.1...v3.0.2) (2023-03-17)



### Chore

* Prepare release v3.0.2


### Docs

* correct spelling of "networks_advanced" (#517) ([#517](https://github.com/kreuzwerker/terraform-provider-docker/issues/517))


### Feat

* add MAC address to state (#523) ([#523](https://github.com/kreuzwerker/terraform-provider-docker/issues/523))


### Fix

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.25.0 (#515) ([#515](https://github.com/kreuzwerker/terraform-provider-docker/issues/515))

* update module github.com/hashicorp/terraform-plugin-docs to v0.14.1 (#519) ([#519](https://github.com/kreuzwerker/terraform-provider-docker/issues/519))

* Implement proxy support. (#529) ([#529](https://github.com/kreuzwerker/terraform-provider-docker/issues/529))


<a name="v3.0.1"></a>
## [v3.0.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v3.0.0...v3.0.1) (2023-01-13)



### Chore

* Prepare release v3.0.1


### Fix

* Access health of container correctly. (#506) ([#506](https://github.com/kreuzwerker/terraform-provider-docker/issues/506))


<a name="v3.0.0"></a>
## [v3.0.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.25.0...v3.0.0) (2023-01-13)



### Chore

* Prepare release v3.0.0


### Docs

* Add migration guide and update README (#502) ([#502](https://github.com/kreuzwerker/terraform-provider-docker/issues/502))

* Update documentation.


### Feat

* Prepare v3 release (#503) ([#503](https://github.com/kreuzwerker/terraform-provider-docker/issues/503))


<a name="v2.25.0"></a>
## [v2.25.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.24.0...v2.25.0) (2023-01-05)



### Chore

* Prepare release v2.25.0


### Docs

* Add documentation of remote hosts. (#498) ([#498](https://github.com/kreuzwerker/terraform-provider-docker/issues/498))


### Feat

* Add sysctl implementation to container of docker_service. (#499) ([#499](https://github.com/kreuzwerker/terraform-provider-docker/issues/499))

* Add platform attribute to docker_image resource (#500) ([#500](https://github.com/kreuzwerker/terraform-provider-docker/issues/500))

* add alias for networks (#241) ([#241](https://github.com/kreuzwerker/terraform-provider-docker/issues/241))

* Migrate build block to `docker_image` (#501) ([#501](https://github.com/kreuzwerker/terraform-provider-docker/issues/501))


<a name="v2.24.0"></a>
## [v2.24.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.23.1...v2.24.0) (2022-12-23)



### Chore

* update goreleaser/goreleaser-action action to v4 (#485) ([#485](https://github.com/kreuzwerker/terraform-provider-docker/issues/485))

* Prepare release v2.24.0


### Docs

* Update command typo (#487) ([#487](https://github.com/kreuzwerker/terraform-provider-docker/issues/487))

* Fix generated website.


### Feat

* add IPAM options block for docker networks (#491) ([#491](https://github.com/kreuzwerker/terraform-provider-docker/issues/491))

* Support registries with disabled auth (#494) ([#494](https://github.com/kreuzwerker/terraform-provider-docker/issues/494))

* Add triggers attribute to docker_registry_image (#496) ([#496](https://github.com/kreuzwerker/terraform-provider-docker/issues/496))

* cgroupns support (#497) ([#497](https://github.com/kreuzwerker/terraform-provider-docker/issues/497))


### Fix

* Pin data source specific tag test to older tag.

* update module github.com/docker/cli to v20.10.22+incompatible (#489) ([#489](https://github.com/kreuzwerker/terraform-provider-docker/issues/489))

* update module github.com/docker/docker to v20.10.22+incompatible (#490) ([#490](https://github.com/kreuzwerker/terraform-provider-docker/issues/490))


### Test

* Add test for parsing auth headers.


<a name="v2.23.1"></a>
## [v2.23.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.23.0...v2.23.1) (2022-11-23)



### Chore

* Prepare release v2.23.1


### Fix

* Set OS_ARCH from GOHOSTOS and GOHOSTARCH (#477) ([#477](https://github.com/kreuzwerker/terraform-provider-docker/issues/477))

* update module github.com/moby/buildkit to v0.10.6 (#478) ([#478](https://github.com/kreuzwerker/terraform-provider-docker/issues/478))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.24.1 (#479) ([#479](https://github.com/kreuzwerker/terraform-provider-docker/issues/479))

* Handle Auth Header Scopes (#482) ([#482](https://github.com/kreuzwerker/terraform-provider-docker/issues/482))

* Update shasum of busybox:1.35.0 tag in test.


<a name="v2.23.0"></a>
## [v2.23.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.22.0...v2.23.0) (2022-11-02)



### Chore

* Prepare release v2.23.0


### Feat

* add docker logs data source (#471) ([#471](https://github.com/kreuzwerker/terraform-provider-docker/issues/471))

* wait container healthy state (#467) ([#467](https://github.com/kreuzwerker/terraform-provider-docker/issues/467))


### Fix

* Correct provider name to match the public registry (#462) ([#462](https://github.com/kreuzwerker/terraform-provider-docker/issues/462))

* Update shasum of busybox:1.35.0 tag

* update module github.com/golangci/golangci-lint to v1.50.0 (#469) ([#469](https://github.com/kreuzwerker/terraform-provider-docker/issues/469))

* update module github.com/moby/buildkit to v0.10.5 (#472) ([#472](https://github.com/kreuzwerker/terraform-provider-docker/issues/472))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.24.0 (#456) ([#456](https://github.com/kreuzwerker/terraform-provider-docker/issues/456))

* Update shasum of busybox:1.35.0 tag in test.

* update module github.com/docker/cli to v20.10.21+incompatible (#457) ([#457](https://github.com/kreuzwerker/terraform-provider-docker/issues/457))

* update module github.com/docker/docker to v20.10.21+incompatible (#459) ([#459](https://github.com/kreuzwerker/terraform-provider-docker/issues/459))


<a name="v2.22.0"></a>
## [v2.22.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.21.0...v2.22.0) (2022-09-20)



### Chore

* Prepare release v2.22.0


### Feat

* Configurable timeout for docker_container resource stateChangeConf (#454) ([#454](https://github.com/kreuzwerker/terraform-provider-docker/issues/454))


### Fix

* oauth authorization support for azurecr (#451) ([#451](https://github.com/kreuzwerker/terraform-provider-docker/issues/451))


<a name="v2.21.0"></a>
## [v2.21.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.20.3...v2.21.0) (2022-09-05)



### Chore

* Prepare release v2.21.0


### Docs

* Fix docker config example.


### Feat

* Update used goversion to 1.18. (#449) ([#449](https://github.com/kreuzwerker/terraform-provider-docker/issues/449))

* Add image_id attribute to docker_image resource. (#450) ([#450](https://github.com/kreuzwerker/terraform-provider-docker/issues/450))


### Fix

* update module github.com/docker/go-units to v0.5.0 (#445) ([#445](https://github.com/kreuzwerker/terraform-provider-docker/issues/445))

* update module github.com/golangci/golangci-lint to v1.49.0 (#441) ([#441](https://github.com/kreuzwerker/terraform-provider-docker/issues/441))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.21.0 (#434) ([#434](https://github.com/kreuzwerker/terraform-provider-docker/issues/434))

* Fix repo_digest value for DockerImageDatasource test.

* Remove reading part of docker_tag resource. (#448) ([#448](https://github.com/kreuzwerker/terraform-provider-docker/issues/448))

* Replace deprecated .latest attribute with new image_id. (#453) ([#453](https://github.com/kreuzwerker/terraform-provider-docker/issues/453))


<a name="v2.20.3"></a>
## [v2.20.3](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.20.2...v2.20.3) (2022-08-31)



### Chore

* Prepare release v2.20.3


### Fix

* Adding Support for Windows Paths in Bash (#438) ([#438](https://github.com/kreuzwerker/terraform-provider-docker/issues/438))

* update module github.com/katbyte/terrafmt to v0.5.2 (#437) ([#437](https://github.com/kreuzwerker/terraform-provider-docker/issues/437))

* Docker Registry Image data source use HEAD request to query image digest (#433) ([#433](https://github.com/kreuzwerker/terraform-provider-docker/issues/433))

* update module github.com/moby/buildkit to v0.10.4 (#440) ([#440](https://github.com/kreuzwerker/terraform-provider-docker/issues/440))


<a name="v2.20.2"></a>
## [v2.20.2](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.20.1...v2.20.2) (2022-08-10)



### Chore

* Prepare release v2.20.2


### Fix

* Check the operating system for determining the default Docker socket (#427) ([#427](https://github.com/kreuzwerker/terraform-provider-docker/issues/427))

* update module github.com/golangci/golangci-lint to v1.48.0 (#423) ([#423](https://github.com/kreuzwerker/terraform-provider-docker/issues/423) [#431](https://github.com/kreuzwerker/terraform-provider-docker/issues/431))


### Other

* Revert "fix(deps): update module github.com/golangci/golangci-lint to v1.48.0 (#423)" ([#423](https://github.com/kreuzwerker/terraform-provider-docker/issues/423))


<a name="v2.20.1"></a>
## [v2.20.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.20.0...v2.20.1) (2022-08-10)



### Chore

* Reduce time to setup AccTests (#430) ([#430](https://github.com/kreuzwerker/terraform-provider-docker/issues/430))

* Prepare release v2.20.1


### Docs

* Improve docker network usage documentation [skip-ci]


### Feat

* Implement triggers attribute for docker_image. (#425) ([#425](https://github.com/kreuzwerker/terraform-provider-docker/issues/425))


### Fix

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.20.0 (#422) ([#422](https://github.com/kreuzwerker/terraform-provider-docker/issues/422))

* update module github.com/golangci/golangci-lint to v1.47.2 (#411) ([#411](https://github.com/kreuzwerker/terraform-provider-docker/issues/411))

* Add ForceTrue to docker_image name attribute. (#421) ([#421](https://github.com/kreuzwerker/terraform-provider-docker/issues/421))

* update module github.com/katbyte/terrafmt to v0.5.1 (#429) ([#429](https://github.com/kreuzwerker/terraform-provider-docker/issues/429))


<a name="v2.20.0"></a>
## [v2.20.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.19.0...v2.20.0) (2022-07-28)



### Chore

* update actions/setup-go action to v3 (#353) ([#353](https://github.com/kreuzwerker/terraform-provider-docker/issues/353))

* update goreleaser/goreleaser-action action to v3 (#389) ([#389](https://github.com/kreuzwerker/terraform-provider-docker/issues/389))

* Fix release targets in Makefile.

* update module go to 1.18 (#412) ([#412](https://github.com/kreuzwerker/terraform-provider-docker/issues/412))

* Prepare release v2.20.0


### Feat

* Implement support for insecure registries (#414) ([#414](https://github.com/kreuzwerker/terraform-provider-docker/issues/414))

* Implementation of `docker_tag` resource. (#418) ([#418](https://github.com/kreuzwerker/terraform-provider-docker/issues/418))


### Fix

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.19.0 (#410) ([#410](https://github.com/kreuzwerker/terraform-provider-docker/issues/410))


<a name="v2.19.0"></a>
## [v2.19.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.18.1...v2.19.0) (2022-07-15)



### Chore

* Prepare release v2.19.0


### Feat

* Add gpu flag to docker_container resource (#405) ([#405](https://github.com/kreuzwerker/terraform-provider-docker/issues/405))


### Fix

* ECR authentication (#409) ([#409](https://github.com/kreuzwerker/terraform-provider-docker/issues/409))

* Enable authentication to multiple registries again. (#400) ([#400](https://github.com/kreuzwerker/terraform-provider-docker/issues/400))


<a name="v2.18.1"></a>
## [v2.18.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.18.0...v2.18.1) (2022-07-14)



### Chore

* update actions/checkout action to v3 (#354) ([#354](https://github.com/kreuzwerker/terraform-provider-docker/issues/354))

* Automate changelog generation [skip ci]

* Prepare release v2.18.1


### Fix

* Enables having a Dockerfile outside the context (#402) ([#402](https://github.com/kreuzwerker/terraform-provider-docker/issues/402))

* Throw errors when any part of docker config file handling goes wrong. (#406) ([#406](https://github.com/kreuzwerker/terraform-provider-docker/issues/406))

* Improve searchLocalImages error handling. (#407) ([#407](https://github.com/kreuzwerker/terraform-provider-docker/issues/407))

* update module github.com/moby/buildkit to v0.10.3 (#394) ([#394](https://github.com/kreuzwerker/terraform-provider-docker/issues/394))


<a name="v2.18.0"></a>
## [v2.18.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.17.0...v2.18.0) (2022-07-11)



### Chore

* prepare release v2.18.0


### Feat

* add runtime, stop_signal and stop_timeout properties to the docker_container resource (#364) ([#364](https://github.com/kreuzwerker/terraform-provider-docker/issues/364))


### Fix

* update module github.com/docker/distribution to v2.8.1 (#348) ([#348](https://github.com/kreuzwerker/terraform-provider-docker/issues/348))

* update module github.com/golangci/golangci-lint to v1.46.2 (#341) ([#341](https://github.com/kreuzwerker/terraform-provider-docker/issues/341))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.18.0 (#396) ([#396](https://github.com/kreuzwerker/terraform-provider-docker/issues/396))

* compare relative paths when excluding, fixes kreuzwerker#280 (#397) ([#280](https://github.com/kreuzwerker/terraform-provider-docker/issues/280) [#397](https://github.com/kreuzwerker/terraform-provider-docker/issues/397))

* Switch to proper go tools mechanism to fix website-* workflows. (#399) ([#399](https://github.com/kreuzwerker/terraform-provider-docker/issues/399))

* Correctly handle build files and context for docker_registry_image (#398) ([#398](https://github.com/kreuzwerker/terraform-provider-docker/issues/398))


<a name="v2.17.0"></a>
## [v2.17.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.16.0...v2.17.0) (2022-06-23)



### Chore

* remove the workflow to close stale issues and pull requests (#371) ([#371](https://github.com/kreuzwerker/terraform-provider-docker/issues/371))

* update golangci/golangci-lint-action action to v3 (#352) ([#352](https://github.com/kreuzwerker/terraform-provider-docker/issues/352))

* split acc test into resources (#382) ([#382](https://github.com/kreuzwerker/terraform-provider-docker/issues/382))

* improve image delete error message (#359) ([#359](https://github.com/kreuzwerker/terraform-provider-docker/issues/359))

* Update website-generation workflow (#386) ([#386](https://github.com/kreuzwerker/terraform-provider-docker/issues/386))

* Exclude examples directory from renovate.

* prepare release v2.17.0


### Feat

* Enable buildkit when client has support. (#387) ([#387](https://github.com/kreuzwerker/terraform-provider-docker/issues/387))


### Fix

* correct authentication for ghcr.io registry(#349) ([#349](https://github.com/kreuzwerker/terraform-provider-docker/issues/349))

* update module github.com/docker/cli to v20.10.17 (#324) ([#324](https://github.com/kreuzwerker/terraform-provider-docker/issues/324))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.17.0 (#357) ([#357](https://github.com/kreuzwerker/terraform-provider-docker/issues/357))

* Pipeline updates (#390) ([#390](https://github.com/kreuzwerker/terraform-provider-docker/issues/390))

* update go package files directly on master to fix build.


<a name="v2.16.0"></a>
## [v2.16.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.15.0...v2.16.0) (2022-01-24)



### Chore

* update alpine docker tag to v3.14.2 (#277) ([#277](https://github.com/kreuzwerker/terraform-provider-docker/issues/277))

* update tj-actions/verify-changed-files action to v7.2 (#285) ([#285](https://github.com/kreuzwerker/terraform-provider-docker/issues/285))

* update nginx:1.21.1 docker digest to a05b0cd (#273) ([#273](https://github.com/kreuzwerker/terraform-provider-docker/issues/273))

* update golang to v1.17 (#272) ([#272](https://github.com/kreuzwerker/terraform-provider-docker/issues/272))

* unify require sections in go.mod files

* update workflows and docs to go 1.17

* update nginx docker tag to v1.21.3 (#287) ([#287](https://github.com/kreuzwerker/terraform-provider-docker/issues/287))

* update nginx:1.21.3 docker digest to 644a705 (#295) ([#295](https://github.com/kreuzwerker/terraform-provider-docker/issues/295))

* update tj-actions/verify-changed-files action to v8 (#299) ([#299](https://github.com/kreuzwerker/terraform-provider-docker/issues/299))

* update tj-actions/verify-changed-files action to v8.3 (#303) ([#303](https://github.com/kreuzwerker/terraform-provider-docker/issues/303))

* update nginx docker tag to v1.21.4 (#309) ([#309](https://github.com/kreuzwerker/terraform-provider-docker/issues/309))

* update tj-actions/verify-changed-files action to v8.8 (#308) ([#308](https://github.com/kreuzwerker/terraform-provider-docker/issues/308))

* prepare release v2.16.0


### Docs

* fix r/registry_image truncated docs (#304) ([#304](https://github.com/kreuzwerker/terraform-provider-docker/issues/304))

* update registry_image.md (#321) ([#321](https://github.com/kreuzwerker/terraform-provider-docker/issues/321))

* fix service options (#337) ([#337](https://github.com/kreuzwerker/terraform-provider-docker/issues/337) [#335](https://github.com/kreuzwerker/terraform-provider-docker/issues/335))


### Feat

* add parameter for SSH options (#335) ([#335](https://github.com/kreuzwerker/terraform-provider-docker/issues/335))


### Fix

* update module github.com/golangci/golangci-lint to v1.42.0 (#274) ([#274](https://github.com/kreuzwerker/terraform-provider-docker/issues/274))

* update module github.com/golangci/golangci-lint to v1.42.1 (#284) ([#284](https://github.com/kreuzwerker/terraform-provider-docker/issues/284))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.7.1 (#279) ([#279](https://github.com/kreuzwerker/terraform-provider-docker/issues/279))

* fmt of go files for go 1.17

* update module github.com/hashicorp/terraform-plugin-docs to v0.5.0 (#286) ([#286](https://github.com/kreuzwerker/terraform-provider-docker/issues/286))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.8.0 (#298) ([#298](https://github.com/kreuzwerker/terraform-provider-docker/issues/298))

* update module github.com/docker/cli to v20.10.10 (#296) ([#296](https://github.com/kreuzwerker/terraform-provider-docker/issues/296))

* Fixed typo (#310) ([#310](https://github.com/kreuzwerker/terraform-provider-docker/issues/310))

* remove log_driver's default value and make log_driver `computed` (#270) ([#270](https://github.com/kreuzwerker/terraform-provider-docker/issues/270))

* add nil check of DriverConfig (#315) ([#315](https://github.com/kreuzwerker/terraform-provider-docker/issues/315))

* update module github.com/docker/docker to v20.10.10 (#297) ([#297](https://github.com/kreuzwerker/terraform-provider-docker/issues/297))

* update module github.com/docker/cli to v20.10.11 (#316) ([#316](https://github.com/kreuzwerker/terraform-provider-docker/issues/316))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.9.0 (#317) ([#317](https://github.com/kreuzwerker/terraform-provider-docker/issues/317))

* update module github.com/hashicorp/terraform-plugin-docs to v0.5.1 (#311) ([#311](https://github.com/kreuzwerker/terraform-provider-docker/issues/311))

* update module github.com/golangci/golangci-lint to v1.43.0 (#306) ([#306](https://github.com/kreuzwerker/terraform-provider-docker/issues/306))

* pass container rm flag (#322) ([#322](https://github.com/kreuzwerker/terraform-provider-docker/issues/322) [#321](https://github.com/kreuzwerker/terraform-provider-docker/issues/321))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.10.1 (#323) ([#323](https://github.com/kreuzwerker/terraform-provider-docker/issues/323))


<a name="v2.15.0"></a>
## [v2.15.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.14.0...v2.15.0) (2021-08-11)



### Chore

* update nginx:1.21.1 docker digest to 353c20f (#248) ([#248](https://github.com/kreuzwerker/terraform-provider-docker/issues/248))

* update actions/stale action to v4 (#250) ([#250](https://github.com/kreuzwerker/terraform-provider-docker/issues/250))

* update nginx:1.21.1 docker digest to 11d4e59 (#251) ([#251](https://github.com/kreuzwerker/terraform-provider-docker/issues/251))

* update nginx:1.21.1 docker digest to 8f33576 (#252) ([#252](https://github.com/kreuzwerker/terraform-provider-docker/issues/252))

* update alpine docker tag to v3.14.1 (#263) ([#263](https://github.com/kreuzwerker/terraform-provider-docker/issues/263))

* adapt acc test docker version (#266) ([#266](https://github.com/kreuzwerker/terraform-provider-docker/issues/266))

* re go gets terraform-plugin-docs

* prepare release v2.15.0


### Docs

* add badges

* corrects authentication misspell. Closes #264 ([#264](https://github.com/kreuzwerker/terraform-provider-docker/issues/264))


### Feat

* add container storage opts (#258) ([#258](https://github.com/kreuzwerker/terraform-provider-docker/issues/258))


### Fix

* update module github.com/docker/cli to v20.10.8 (#255) ([#255](https://github.com/kreuzwerker/terraform-provider-docker/issues/255))

* update module github.com/docker/docker to v20.10.8 (#256) ([#256](https://github.com/kreuzwerker/terraform-provider-docker/issues/256))

* add current timestamp for file upload to container (#259) ([#259](https://github.com/kreuzwerker/terraform-provider-docker/issues/259))


<a name="v2.14.0"></a>
## [v2.14.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.13.0...v2.14.0) (2021-07-09)



### Chore

* update nginx:1.21.0 docker digest to d1b8ff2 (#232) ([#232](https://github.com/kreuzwerker/terraform-provider-docker/issues/232))

* update tj-actions/verify-changed-files action to v7 (#237) ([#237](https://github.com/kreuzwerker/terraform-provider-docker/issues/237))

* update nginx docker tag to v1.21.1 (#243) ([#243](https://github.com/kreuzwerker/terraform-provider-docker/issues/243))

* prepare release v2.14.0


### Docs

* update readme with logos and subsections (#235) ([#235](https://github.com/kreuzwerker/terraform-provider-docker/issues/235))

* docs/service entrypoint (#244) ([#244](https://github.com/kreuzwerker/terraform-provider-docker/issues/244))

* update to absolute path for registry image context (#246) ([#246](https://github.com/kreuzwerker/terraform-provider-docker/issues/246))


### Feat

* support terraform v1 (#242) ([#242](https://github.com/kreuzwerker/terraform-provider-docker/issues/242))


### Fix

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.7.0 (#236) ([#236](https://github.com/kreuzwerker/terraform-provider-docker/issues/236))

* fix/service bind options (#234) ([#234](https://github.com/kreuzwerker/terraform-provider-docker/issues/234))

* Update the URL of the docker hub registry (#230) ([#230](https://github.com/kreuzwerker/terraform-provider-docker/issues/230))

* consider .dockerignore in image build (#240) ([#240](https://github.com/kreuzwerker/terraform-provider-docker/issues/240))


<a name="v2.13.0"></a>
## [v2.13.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.12.2...v2.13.0) (2021-06-22)



### Chore

* chore/refactor tests (#201) ([#201](https://github.com/kreuzwerker/terraform-provider-docker/issues/201))

* update alpine docker tag to v3.13.5

* update nginx:1.18.0-alpine docker digest to 93baf2e

* update nginx:1.18.0-alpine docker digest to 93baf2e

* update nginx docker tag to v1.21.0

* update nginx docker tag to v1.21.0

* update stocard/gotthard docker digest to 38c2216

* update stocard/gotthard docker digest to 38c2216

* update alpine docker tag to v3.14.0 (#225) ([#225](https://github.com/kreuzwerker/terraform-provider-docker/issues/225))

* prepare release v2.13.0


### Docs

* fix typos in docker_image example usage (#213) ([#213](https://github.com/kreuzwerker/terraform-provider-docker/issues/213))

* fix a few typos (#216) ([#216](https://github.com/kreuzwerker/terraform-provider-docker/issues/216))

* add oneline cli git cmd to get the changes since the last tag


### Fix

* update module github.com/golangci/golangci-lint to v1.40.1 (#194) ([#194](https://github.com/kreuzwerker/terraform-provider-docker/issues/194))

* update module github.com/docker/cli to v20.10.7 (#217) ([#217](https://github.com/kreuzwerker/terraform-provider-docker/issues/217))

* fix/service image name (#212) ([#212](https://github.com/kreuzwerker/terraform-provider-docker/issues/212))

* update module github.com/golangci/golangci-lint to v1.41.1 (#226) ([#226](https://github.com/kreuzwerker/terraform-provider-docker/issues/226))

* update module github.com/docker/docker to v20.10.7 (#218) ([#218](https://github.com/kreuzwerker/terraform-provider-docker/issues/218))

* fix/service delete deadline (#227) ([#227](https://github.com/kreuzwerker/terraform-provider-docker/issues/227))


### Other

* include image name in error message (#223) ([#223](https://github.com/kreuzwerker/terraform-provider-docker/issues/223))


<a name="v2.12.2"></a>
## [v2.12.2](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.12.1...v2.12.2) (2021-05-26)



### Chore

* prepare release v2.12.2


<a name="v2.12.1"></a>
## [v2.12.1](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.12.0...v2.12.1) (2021-05-26)



### Chore

* update tj-actions/verify-changed-files action to v6.2 (#200) ([#200](https://github.com/kreuzwerker/terraform-provider-docker/issues/200))

* update changelog for v2.12.1


### Fix

* service state upgradeV2 for empty auth ([#203](https://github.com/kreuzwerker/terraform-provider-docker/issues/203))

* add service host flattener with space split (#205) ([#205](https://github.com/kreuzwerker/terraform-provider-docker/issues/205))


<a name="v2.12.0"></a>
## [v2.12.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.11.0...v2.12.0) (2021-05-23)



### Chore

* bump docker dependency to v20.10.5 (#119) ([#119](https://github.com/kreuzwerker/terraform-provider-docker/issues/119))

* add the guide about Terraform Configuration in Bug Report (#139) ([#139](https://github.com/kreuzwerker/terraform-provider-docker/issues/139))

* configure actions/stale (#157) ([#157](https://github.com/kreuzwerker/terraform-provider-docker/issues/157))

* configure Renovate (#162) ([#162](https://github.com/kreuzwerker/terraform-provider-docker/issues/162))

* ignore dist folder

* update changelog for v2.12.0


### Ci

* run acceptance tests with multiple Terraform versions (#129) ([#129](https://github.com/kreuzwerker/terraform-provider-docker/issues/129))


### Docs

* fix Github repository URL in README (#136) ([#136](https://github.com/kreuzwerker/terraform-provider-docker/issues/136))

* add a guide about writing issues to CONTRIBUTING.md (#149) ([#149](https://github.com/kreuzwerker/terraform-provider-docker/issues/149))

* update example usage

* add an example to build an image with docker_image (#158) ([#158](https://github.com/kreuzwerker/terraform-provider-docker/issues/158))

* format `Guide of Bug report` (#159) ([#159](https://github.com/kreuzwerker/terraform-provider-docker/issues/159))

* add releasing steps

* update for v2.12.0


### Feat

* migrate to terraform-sdk v2 (#102) ([#102](https://github.com/kreuzwerker/terraform-provider-docker/issues/102))

* support darwin arm builds and golang 1.16 (#140) ([#140](https://github.com/kreuzwerker/terraform-provider-docker/issues/140))

* feat/doc generation (#193) ([#193](https://github.com/kreuzwerker/terraform-provider-docker/issues/193))


### Fix

* set "ForceNew: true" to labelSchema (#152) ([#152](https://github.com/kreuzwerker/terraform-provider-docker/issues/152))

* search local images with Docker image ID (#151) ([#151](https://github.com/kreuzwerker/terraform-provider-docker/issues/151))

* fix/workflows (#169) ([#169](https://github.com/kreuzwerker/terraform-provider-docker/issues/169))

* update module github.com/golangci/golangci-lint to v1.39.0 (#166) ([#166](https://github.com/kreuzwerker/terraform-provider-docker/issues/166))

* update module github.com/katbyte/terrafmt to v0.3.0 (#168) ([#168](https://github.com/kreuzwerker/terraform-provider-docker/issues/168))

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.5.0 (#167) ([#167](https://github.com/kreuzwerker/terraform-provider-docker/issues/167))

* update module github.com/docker/docker to v20.10.5 (#165) ([#165](https://github.com/kreuzwerker/terraform-provider-docker/issues/165))

* update module github.com/docker/cli to v20.10.5 (#164) ([#164](https://github.com/kreuzwerker/terraform-provider-docker/issues/164))

* update module github.com/docker/docker to v20.10.6 (#174) ([#174](https://github.com/kreuzwerker/terraform-provider-docker/issues/174))

* update module github.com/docker/cli to v20.10.6 (#175) ([#175](https://github.com/kreuzwerker/terraform-provider-docker/issues/175))

* fix/move helpers (#170) ([#170](https://github.com/kreuzwerker/terraform-provider-docker/issues/170))

* assign map to rawState when it is nil to prevent panic (#180) ([#180](https://github.com/kreuzwerker/terraform-provider-docker/issues/180))

* remove gpg key from action to make it work in forks

* skip sign on compile action

* update module github.com/hashicorp/terraform-plugin-sdk/v2 to v2.6.1 (#181) ([#181](https://github.com/kreuzwerker/terraform-provider-docker/issues/181))

* replace for loops with StateChangeConf (#182) ([#182](https://github.com/kreuzwerker/terraform-provider-docker/issues/182))

* update module github.com/golangci/golangci-lint to v1.40.0 (#191) ([#191](https://github.com/kreuzwerker/terraform-provider-docker/issues/191))

* test spaces for windows (#190) ([#190](https://github.com/kreuzwerker/terraform-provider-docker/issues/190))

* rewriting tar header fields (#198) ([#198](https://github.com/kreuzwerker/terraform-provider-docker/issues/198) [#192](https://github.com/kreuzwerker/terraform-provider-docker/issues/192))


### Other

* Merge branch 'master' of github.com:kreuzwerker/terraform-provider-docker


<a name="v2.11.0"></a>
## [v2.11.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.9.0...v2.11.0) (2021-01-22)



### Chore

* updates changelog for v2.10.0

* update changelog for v2.11.0


### Ci

* pins workflows to ubuntu:20.04 image

* bumps to docker version 20.10.1


### Docs

* updates links to provider and slack

* fixes broken links

* fixes more broken links

* fixes copy paster

* add labels to arguments of docker_service (#105) ([#105](https://github.com/kreuzwerker/terraform-provider-docker/issues/105) [#103](https://github.com/kreuzwerker/terraform-provider-docker/issues/103))

* fix legacy configuration style (#126) ([#126](https://github.com/kreuzwerker/terraform-provider-docker/issues/126))


### Feat

* add ability to lint/check of links in documentation locally (#98) ([#98](https://github.com/kreuzwerker/terraform-provider-docker/issues/98))

* add local semantic commit validation (#99) ([#99](https://github.com/kreuzwerker/terraform-provider-docker/issues/99))

* add force_remove option to r/image (#104) ([#104](https://github.com/kreuzwerker/terraform-provider-docker/issues/104))

* support max replicas of Docker Service Task Spec (#112) ([#112](https://github.com/kreuzwerker/terraform-provider-docker/issues/112) [#111](https://github.com/kreuzwerker/terraform-provider-docker/issues/111))

* supports Docker plugin (#35) ([#35](https://github.com/kreuzwerker/terraform-provider-docker/issues/35) [#24](https://github.com/kreuzwerker/terraform-provider-docker/issues/24))

* add properties -it (tty and stdin_opn) to docker container ([#120](https://github.com/kreuzwerker/terraform-provider-docker/issues/120))


### Fix

* image label for workflows

* adds issues path in links

* set "latest" to tag when tag isn't specified (#117) ([#117](https://github.com/kreuzwerker/terraform-provider-docker/issues/117))


### Other

* Merge branch 'master' into chore-gh-issue-tpl

* Merge pull request #36 from kreuzwerker/chore-gh-issue-tpl ([#36](https://github.com/kreuzwerker/terraform-provider-docker/issues/36))

* Merge pull request #38 from kreuzwerker/ci-ubuntu2004-workflow ([#38](https://github.com/kreuzwerker/terraform-provider-docker/issues/38))

* aligns braces


<a name="v2.9.0"></a>
## [v2.9.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/v2.8.0...v2.9.0) (2020-12-25)



### Chore

* fix changelog links

* introduces golangci-lint (#32) ([#32](https://github.com/kreuzwerker/terraform-provider-docker/issues/32) [#13](https://github.com/kreuzwerker/terraform-provider-docker/issues/13))

* update changelog 2.8.0 release date

* adds separate bug and ft req templates

* ignores testing folders

* updates changelog for 2.9.0


### Ci

* fix test of website

* remove unneeded make tasks

* add gofmt's '-s' option


### Docs

* devices is a block, not a boolean

* adds coc and contributing

* cleans readme


### Feat

* adds support for init process injection for containers. (#300) ([#300](https://github.com/kreuzwerker/terraform-provider-docker/issues/300))

* adds security_opts to container config. (#308) ([#308](https://github.com/kreuzwerker/terraform-provider-docker/issues/308) [#288](https://github.com/kreuzwerker/terraform-provider-docker/issues/288))

* adds support for OCI manifests (#316) ([#316](https://github.com/kreuzwerker/terraform-provider-docker/issues/316) [#315](https://github.com/kreuzwerker/terraform-provider-docker/issues/315))


### Fix

* workdir null behavior (#320) ([#320](https://github.com/kreuzwerker/terraform-provider-docker/issues/320) [#319](https://github.com/kreuzwerker/terraform-provider-docker/issues/319))

* treat null user as a no-op (#318) ([#318](https://github.com/kreuzwerker/terraform-provider-docker/issues/318) [#317](https://github.com/kreuzwerker/terraform-provider-docker/issues/317))

* allow healthcheck to be computed as container can specify (#312) ([#312](https://github.com/kreuzwerker/terraform-provider-docker/issues/312))

* changing mounts requires ForceNew (#314) ([#314](https://github.com/kreuzwerker/terraform-provider-docker/issues/314))

* Fix crash

* remove all azure cps


### Other

* Merge branch 'master' of github.com:terraform-providers/terraform-provider-docker

* Merge branch 'master' of github.com:terraform-providers/terraform-provider-docker

* Merge branch 'master' of github.com:terraform-providers/terraform-provider-docker

* Merge branch 'master' of github.com:terraform-providers/terraform-provider-docker

* Merge pull request #8 from dubo-dubon-duponey/patch1 ([#8](https://github.com/kreuzwerker/terraform-provider-docker/issues/8))

* Merge pull request #26 from kreuzwerker/ci/fix-website-ci ([#26](https://github.com/kreuzwerker/terraform-provider-docker/issues/26))

* format with gofumpt

* Merge pull request #11 from suzuki-shunsuke/format-with-gofumpt ([#11](https://github.com/kreuzwerker/terraform-provider-docker/issues/11))

* Merge pull request #33 from brandonros/patch-1 ([#33](https://github.com/kreuzwerker/terraform-provider-docker/issues/33))


<a name="v2.8.0"></a>
## [v2.8.0](https://github.com/kreuzwerker/terraform-provider-docker/compare/...v2.8.0) (2020-11-11)



### Chore

* adapts cp debug outputs

* fix typo (#292) ([#292](https://github.com/kreuzwerker/terraform-provider-docker/issues/292))

* updates link syntax (#287) ([#287](https://github.com/kreuzwerker/terraform-provider-docker/issues/287))

* documentation updates (#286) ([#286](https://github.com/kreuzwerker/terraform-provider-docker/issues/286))

* bump go 115 (#297) ([#297](https://github.com/kreuzwerker/terraform-provider-docker/issues/297))

* removes vendor dir (#298) ([#298](https://github.com/kreuzwerker/terraform-provider-docker/issues/298))

* deactivates travis

* removes travis.yml

* updates changelog for 2.8.0


### Ci

* skips test which is flaky only on travis

* removes debug option from acc tests

* bumps docker and ubuntu versions (#241) ([#241](https://github.com/kreuzwerker/terraform-provider-docker/issues/241))

* adds goreleaser and gh action

* adds unit test workflow

* enables unit tests for master branch

* adds go version and goproxy env

* adds compile

* adds acc test

* adds website test to unit test

* fix workflow names

* make website check separate workflow

* fix gopath

* switches from free to openbsd

* fix absolute gopath for website

* exports gopath manually

* consolidates all cmds into 1

* only run website workflow

* fix website

* reactivats all workflows


### Docs

* improve validation of runtime constraints

* provider/docker - network settings attrs

* Document new data source + limitations (#7814) ([#7814](https://github.com/kreuzwerker/terraform-provider-docker/issues/7814))

* Fix example for docker_registry_image (#8308) ([#8308](https://github.com/kreuzwerker/terraform-provider-docker/issues/8308))

* Fix exported attribute name in docker_registry_image

* Fix misspelled words

* docker_container resource mounts support (#147) ([#147](https://github.com/kreuzwerker/terraform-provider-docker/issues/147))

* docker_network data source (#87). Closes #84 ([#87](https://github.com/kreuzwerker/terraform-provider-docker/issues/87) [#84](https://github.com/kreuzwerker/terraform-provider-docker/issues/84))

* update anchors with -1 suffix (#178) ([#178](https://github.com/kreuzwerker/terraform-provider-docker/issues/178) [#176](https://github.com/kreuzwerker/terraform-provider-docker/issues/176))

* adds new label structure. Closes #214 ([#214](https://github.com/kreuzwerker/terraform-provider-docker/issues/214))

* update restart_policy for service. Closes #228 ([#228](https://github.com/kreuzwerker/terraform-provider-docker/issues/228))

* update service.html.markdown (#281) ([#281](https://github.com/kreuzwerker/terraform-provider-docker/issues/281))

* update container.html.markdown (#278) ([#278](https://github.com/kreuzwerker/terraform-provider-docker/issues/278))


### Feat

* Add Issue Template

* Added Docker links support to the docker_container resource.

* Add docker container network settings to output attribute

* Added support for more complexly images repos such as images on a private registry that are stored as namespace/name

* Add privileged option to docker container resource

* Add network_mode support to docker

* Add support of custom networks in docker

* Add the networks entry

* Add Elem and Set to the network set

* Add `destroy_grace_seconds` option to stop container before delete (#7513) ([#7513](https://github.com/kreuzwerker/terraform-provider-docker/issues/7513))

* Allow Windows Docker containers to map volumes (#13584) ([#13584](https://github.com/kreuzwerker/terraform-provider-docker/issues/13584))

* Added Docker links support to the docker_container resource.

* Add privileged option to docker container resource

* Add network_mode support to docker

* Add support of custom networks in docker

* Add the networks entry

* Add `destroy_grace_seconds` option to stop container before delete (#7513) ([#7513](https://github.com/kreuzwerker/terraform-provider-docker/issues/7513))

* Adding back the GNUmakefile test-compile step

* Feat/private registry support (#21) ([#21](https://github.com/kreuzwerker/terraform-provider-docker/issues/21))

* Allow the awslogs log driver (#28) ([#28](https://github.com/kreuzwerker/terraform-provider-docker/issues/28))

* Feat/swarm 1 update deps (#37) ([#37](https://github.com/kreuzwerker/terraform-provider-docker/issues/37))

* Feat/swarm 2 refactorings (#38) ([#38](https://github.com/kreuzwerker/terraform-provider-docker/issues/38))

* Feat/swarm 3 acc test infra (#39) ([#39](https://github.com/kreuzwerker/terraform-provider-docker/issues/39))

* Add support to attach devices to containers (#54) ([#54](https://github.com/kreuzwerker/terraform-provider-docker/issues/54))

* Add ability to upload executable files (#55) ([#55](https://github.com/kreuzwerker/terraform-provider-docker/issues/55))

* Add Ulimits to containers (#35) ([#35](https://github.com/kreuzwerker/terraform-provider-docker/issues/35))

* Feat/swarm 4 new resources (#40) ([#40](https://github.com/kreuzwerker/terraform-provider-docker/issues/40))

* Adds warning about the link feature

* Add support for running tests on Windows (#90) ([#90](https://github.com/kreuzwerker/terraform-provider-docker/issues/90))

* Adds pid and namespace mode (#96) ([#96](https://github.com/kreuzwerker/terraform-provider-docker/issues/96) [#88](https://github.com/kreuzwerker/terraform-provider-docker/issues/88) [#17](https://github.com/kreuzwerker/terraform-provider-docker/issues/17))

* Add labels to support docker stacks (#92) ([#92](https://github.com/kreuzwerker/terraform-provider-docker/issues/92))

* Add rm and attach options to execute short-lived containers (#106) ([#106](https://github.com/kreuzwerker/terraform-provider-docker/issues/106) [#43](https://github.com/kreuzwerker/terraform-provider-docker/issues/43))

* Adds asc sorting to container ports flattening to fix blinking test.

* Add container healthcheck

* Adds container healthcheck.

* Add capability to not start container (create only)

* Add test

* Adds the docker container start flag. Closes #62. ([#62](https://github.com/kreuzwerker/terraform-provider-docker/issues/62))

* Adds cpu_set to containers. Closes #41 ([#41](https://github.com/kreuzwerker/terraform-provider-docker/issues/41))

* Adds container static IPv4/IPv6 address. Marks network and network_alias as deprecated. Closes #105. ([#105](https://github.com/kreuzwerker/terraform-provider-docker/issues/105))

* Add logs attribute to get container logs when attach option is enabled

* Adds container logs option. Closes #108. ([#108](https://github.com/kreuzwerker/terraform-provider-docker/issues/108))

* Add ssh protocol (copy) (#153). Closes #112 ([#153](https://github.com/kreuzwerker/terraform-provider-docker/issues/153) [#112](https://github.com/kreuzwerker/terraform-provider-docker/issues/112))

* Adds cross-platform support for generic Docker credential helper. (#159) ([#159](https://github.com/kreuzwerker/terraform-provider-docker/issues/159) [#143](https://github.com/kreuzwerker/terraform-provider-docker/issues/143))

* Add support for sysctls (#172) ([#172](https://github.com/kreuzwerker/terraform-provider-docker/issues/172))

* adds container working dir (#181) ([#181](https://github.com/kreuzwerker/terraform-provider-docker/issues/181) [#146](https://github.com/kreuzwerker/terraform-provider-docker/issues/146))

* add container ipc mode. (#182) ([#182](https://github.com/kreuzwerker/terraform-provider-docker/issues/182) [#12](https://github.com/kreuzwerker/terraform-provider-docker/issues/12))

* Add support for group-add (#192) ([#192](https://github.com/kreuzwerker/terraform-provider-docker/issues/192) [#191](https://github.com/kreuzwerker/terraform-provider-docker/issues/191))

* Add `shm_size' attribute for `docker_container' resource. (#190) ([#190](https://github.com/kreuzwerker/terraform-provider-docker/issues/190) [#164](https://github.com/kreuzwerker/terraform-provider-docker/issues/164))

* Add support for readonly containers (#206) ([#206](https://github.com/kreuzwerker/terraform-provider-docker/issues/206) [#203](https://github.com/kreuzwerker/terraform-provider-docker/issues/203))

* adds import for resources (#196) ([#196](https://github.com/kreuzwerker/terraform-provider-docker/issues/196) [#99](https://github.com/kreuzwerker/terraform-provider-docker/issues/99))

* make UID, GID, & mode for secrets and configs configurable (#231) ([#231](https://github.com/kreuzwerker/terraform-provider-docker/issues/231) [#216](https://github.com/kreuzwerker/terraform-provider-docker/issues/216))

* adds config file content as plain string (#232) ([#232](https://github.com/kreuzwerker/terraform-provider-docker/issues/232) [#224](https://github.com/kreuzwerker/terraform-provider-docker/issues/224))

* support to import some docker_container's attributes (#234) ([#234](https://github.com/kreuzwerker/terraform-provider-docker/issues/234))

* supports to update docker_container (#236) ([#236](https://github.com/kreuzwerker/terraform-provider-docker/issues/236))

* allow use of source file instead of content / content_base64 (#240) ([#240](https://github.com/kreuzwerker/terraform-provider-docker/issues/240) [#239](https://github.com/kreuzwerker/terraform-provider-docker/issues/239))

* adds complete support for Docker credential helpers (#253) ([#253](https://github.com/kreuzwerker/terraform-provider-docker/issues/253) [#252](https://github.com/kreuzwerker/terraform-provider-docker/issues/252))

* adds docker Image build feature (#283) ([#283](https://github.com/kreuzwerker/terraform-provider-docker/issues/283))

* conditionally adding port binding (#293). ([#293](https://github.com/kreuzwerker/terraform-provider-docker/issues/293) [#255](https://github.com/kreuzwerker/terraform-provider-docker/issues/255))

* Expose IPv6 properties as attributes


### Fix

* Fix a serious problem when using links.

* Fix Repository attribute in docker client PullOptions for private registries.

* fix resource constraint specs

* Fixing yet more gofmt errors with imports

* Fix typo

* Fix docker test assertions regarding latest tag

* Fix Image Destroy bug. #3609 #3771 ([#3609](https://github.com/kreuzwerker/terraform-provider-docker/issues/3609) [#3771](https://github.com/kreuzwerker/terraform-provider-docker/issues/3771))

* Fix(docs) Correct spelling error in Docker documentation

* Fix #1402 ([#1402](https://github.com/kreuzwerker/terraform-provider-docker/issues/1402))

* Fix Changelog Links Script for docker provider

* Fixing link in README

* Fixing logo

* Fixes layout and notes in docs. (#26) ([#26](https://github.com/kreuzwerker/terraform-provider-docker/issues/26))

* Fixing build and private image

* Fixed test for ulimits on containers

* Fix service flatteners (#66) ([#66](https://github.com/kreuzwerker/terraform-provider-docker/issues/66))

* Fix/map to slice expander (#82) ([#82](https://github.com/kreuzwerker/terraform-provider-docker/issues/82))

* Fix/cert material (#91) ([#91](https://github.com/kreuzwerker/terraform-provider-docker/issues/91) [#86](https://github.com/kreuzwerker/terraform-provider-docker/issues/86) [#14](https://github.com/kreuzwerker/terraform-provider-docker/issues/14))

* fixes ports on containers (#95) ([#95](https://github.com/kreuzwerker/terraform-provider-docker/issues/95) [#8](https://github.com/kreuzwerker/terraform-provider-docker/issues/8) [#89](https://github.com/kreuzwerker/terraform-provider-docker/issues/89) [#73](https://github.com/kreuzwerker/terraform-provider-docker/issues/73))

* Fix other tests as port internal/external are not strings but int

* Fixes bug introduced from merge of #106 ([#106](https://github.com/kreuzwerker/terraform-provider-docker/issues/106))

* Fixes dependencies for old docker client and tests introduced by merge of #49. ([#49](https://github.com/kreuzwerker/terraform-provider-docker/issues/49))

* Fix incorrect indentation for container in docs (#126) ([#126](https://github.com/kreuzwerker/terraform-provider-docker/issues/126))

* Fix container ports issue for asc ordering (#115) ([#115](https://github.com/kreuzwerker/terraform-provider-docker/issues/115))

* Normalize a blank string to 0.0.0.0 (#128) ([#128](https://github.com/kreuzwerker/terraform-provider-docker/issues/128))

* Fix syntax error in docker_service example and make all examples adhere to terraform fmt (#137) ([#137](https://github.com/kreuzwerker/terraform-provider-docker/issues/137))

* Fixes for image pulling and local registry (#143) ([#143](https://github.com/kreuzwerker/terraform-provider-docker/issues/143) [#24](https://github.com/kreuzwerker/terraform-provider-docker/issues/24) [#120](https://github.com/kreuzwerker/terraform-provider-docker/issues/120) [#77](https://github.com/kreuzwerker/terraform-provider-docker/issues/77) [#125](https://github.com/kreuzwerker/terraform-provider-docker/issues/125))

* Fixes for flaky tests (#154) ([#154](https://github.com/kreuzwerker/terraform-provider-docker/issues/154))

* fix website / containers / mount vs mounts (#162) ([#162](https://github.com/kreuzwerker/terraform-provider-docker/issues/162))

* Fix no-op in container when all 'ports' blocks are deleted. (#168) ([#168](https://github.com/kreuzwerker/terraform-provider-docker/issues/168) [#167](https://github.com/kreuzwerker/terraform-provider-docker/issues/167))

* destroy_grace_seconds are considered (#179) ([#179](https://github.com/kreuzwerker/terraform-provider-docker/issues/179) [#174](https://github.com/kreuzwerker/terraform-provider-docker/issues/174))

* service env truncation for multiple delimiters (#193) ([#193](https://github.com/kreuzwerker/terraform-provider-docker/issues/193) [#121](https://github.com/kreuzwerker/terraform-provider-docker/issues/121))

* binary upload as base 64 content (#194) ([#194](https://github.com/kreuzwerker/terraform-provider-docker/issues/194) [#48](https://github.com/kreuzwerker/terraform-provider-docker/issues/48))

* fix service test mount set key

* label for network and volume after improt

* fix sprintf formatter

* replica to 0 in current schema. Closes #221 ([#221](https://github.com/kreuzwerker/terraform-provider-docker/issues/221))

* ports flattening (#233) ([#233](https://github.com/kreuzwerker/terraform-provider-docker/issues/233) [#222](https://github.com/kreuzwerker/terraform-provider-docker/issues/222))

* corrects IPAM config read on the data provider (#229) ([#229](https://github.com/kreuzwerker/terraform-provider-docker/issues/229))

* service endpoint spec flattening

* prevent force recreate of container about some attributes (#269) ([#269](https://github.com/kreuzwerker/terraform-provider-docker/issues/269) [#242](https://github.com/kreuzwerker/terraform-provider-docker/issues/242) [#270](https://github.com/kreuzwerker/terraform-provider-docker/issues/270))

* pins docker registry for tests to v2.7.0

* panic to migrate schema of docker_container from v1 to v2 (#271). Closes #264 ([#271](https://github.com/kreuzwerker/terraform-provider-docker/issues/271) [#264](https://github.com/kreuzwerker/terraform-provider-docker/issues/264))

* port objects with the same internal port but different protocol trigger recreation of container (#274) ([#274](https://github.com/kreuzwerker/terraform-provider-docker/issues/274))

* duplicated buildImage function

* ignores 'remove_volumes' on container import


### Other

* initial commit

* Initial commit. This adds the initial bits of a Docker provider.

* support DOCKER_CERT_PATH

* docker_image acceptance test

* container acceptance tests

* cache client

* make container test better

* ping docker server on startup

* default cert_path to non-nil so input isn't asked

* guard against nil NetworkSettings

* fmt on container resource

* update image sha

* This puts the image parsing code (mostly) back to how it was before. The

* fmt

* When linking to other containers, introduce a slight delay; this lets

* As discussed on the issue, remove the hard-coded delay on startup in

* [tests] change images

* Gofmt change for resource docker_image test

* removed extra parentheses

* fix image test

* entrypoint support for docker_container resource

* restart policy support for docker_container

* add basic runtime constraints to docker_container

* add label support to docker container resource

* support for log driver + config in docker container

* include hostconfig when creating docker_container

* Merge pull request #3761 from ryane/f-provider-docker-improvements ([#3761](https://github.com/kreuzwerker/terraform-provider-docker/issues/3761))

* Refer to a tag instead of latest

* locate container via ID not name ([#3364](https://github.com/kreuzwerker/terraform-provider-docker/issues/3364))

* Convert v to string

* Merge branch 'docker_network' of https://github.com/ColinHebert/terraform into ColinHebert-docker_network

* Fix flaky integration tests

* Add hosts parameter for containers

* Inline ports and volumes schemas for consistency

* Merge branch 'docker-extra-hosts' of https://github.com/paulbellamy/terraform into paulbellamy-docker-extra-hosts

* Tweak and test `host_entry`

* Add `docker_volume` resource

* Mount named volumes in containers

* Catch potential custom network errors in docker

* remove extra parenthesis

* Change default DOCKER_HOST value, fixes #4923 ([#4923](https://github.com/kreuzwerker/terraform-provider-docker/issues/4923))

* Use built-in schema.HashString.

* Merge pull request #5046 from tpounds/use-built-in-schema-string-hash ([#5046](https://github.com/kreuzwerker/terraform-provider-docker/issues/5046))

* Stop providing the hostConfig while starting the container

* #2417 Add support for restart policy unless-stopped ([#2417](https://github.com/kreuzwerker/terraform-provider-docker/issues/2417))

* #5298 Add support for docker run --user option ([#5298](https://github.com/kreuzwerker/terraform-provider-docker/issues/5298))

* Provider Docker: (#6376) ([#6376](https://github.com/kreuzwerker/terraform-provider-docker/issues/6376))

* don't crash with empty commands ([#6409](https://github.com/kreuzwerker/terraform-provider-docker/issues/6409))

* Fixing the Docker Container Mount Test

* Docker DNS Setting Enhancements (#7392) ([#7392](https://github.com/kreuzwerker/terraform-provider-docker/issues/7392))

* Docker documentation and additional test message (#7412) ([#7412](https://github.com/kreuzwerker/terraform-provider-docker/issues/7412))

* Added docker_registry_image data source (#7000) ([#7000](https://github.com/kreuzwerker/terraform-provider-docker/issues/7000))

* Fixes for docker_container host object and documentation (#9367) ([#9367](https://github.com/kreuzwerker/terraform-provider-docker/issues/9367) [#9350](https://github.com/kreuzwerker/terraform-provider-docker/issues/9350) [#9350](https://github.com/kreuzwerker/terraform-provider-docker/issues/9350))

* authentication via values instead of files (#10151) ([#10151](https://github.com/kreuzwerker/terraform-provider-docker/issues/10151))

* Upload files into container before first start (#9520) ([#9520](https://github.com/kreuzwerker/terraform-provider-docker/issues/9520))

* fix regression, cert_path stop working (#10754) (#10801) ([#10754](https://github.com/kreuzwerker/terraform-provider-docker/issues/10754) [#10801](https://github.com/kreuzwerker/terraform-provider-docker/issues/10801))

* Add network create --internal flag support (#10932) ([#10932](https://github.com/kreuzwerker/terraform-provider-docker/issues/10932))

* Add support for a list of pull_triggers within the docker_image resource. (#10845) ([#10845](https://github.com/kreuzwerker/terraform-provider-docker/issues/10845))

* added support for linux capabilities (#12045) ([#12045](https://github.com/kreuzwerker/terraform-provider-docker/issues/12045) [#11623](https://github.com/kreuzwerker/terraform-provider-docker/issues/11623))

* provider/docker network alias (#14710) ([#14710](https://github.com/kreuzwerker/terraform-provider-docker/issues/14710))

* Transfer docker provider

* Merge branch 'master' of /Users/jake/terraform

* Initial transfer of provider code

* Updating Makefile + Add gitignore

* docker docs

* note on docker

* Merge pull request #1564 from nickryand/docker_links ([#1564](https://github.com/kreuzwerker/terraform-provider-docker/issues/1564))

* The docker-image resource expects name, not image

* Update container.html.markdown

* Example for the command arg on docker_container. ([#3011](https://github.com/kreuzwerker/terraform-provider-docker/issues/3011))

* Merge pull request #3383 from apparentlymart/docker-container-command-docs ([#3383](https://github.com/kreuzwerker/terraform-provider-docker/issues/3383))

* entrypoint support for docker_container resource

* restart policy support for docker_container

* add basic runtime constraints to docker_container

* add label support to docker container resource

* support for log driver + config in docker container

* Merge branch 'docker_network' of https://github.com/ColinHebert/terraform into ColinHebert-docker_network

* Add hosts parameter for containers

* Merge branch 'docker-extra-hosts' of https://github.com/paulbellamy/terraform into paulbellamy-docker-extra-hosts

* Tweak and test `host_entry`

* Add `docker_volume` resource

* Mount named volumes in containers

* Update documentation

* #2417 Add support for restart policy unless-stopped ([#2417](https://github.com/kreuzwerker/terraform-provider-docker/issues/2417))

* #5298 Add support for docker run --user option ([#5298](https://github.com/kreuzwerker/terraform-provider-docker/issues/5298))

* Provider Docker: (#6376) ([#6376](https://github.com/kreuzwerker/terraform-provider-docker/issues/6376))

* Docker documentation and additional test message (#7412) ([#7412](https://github.com/kreuzwerker/terraform-provider-docker/issues/7412))

* Docs sweep for lists & maps

* Update index.html.markdown

* Update index.html.markdown

* Add alternative to cert_path

* Fixes for docker_container host object and documentation (#9367) ([#9367](https://github.com/kreuzwerker/terraform-provider-docker/issues/9367) [#9350](https://github.com/kreuzwerker/terraform-provider-docker/issues/9350) [#9350](https://github.com/kreuzwerker/terraform-provider-docker/issues/9350))

* authentication via values instead of files (#10151) ([#10151](https://github.com/kreuzwerker/terraform-provider-docker/issues/10151))

* Upload files into container before first start (#9520) ([#9520](https://github.com/kreuzwerker/terraform-provider-docker/issues/9520))

* Add network create --internal flag support (#10932) ([#10932](https://github.com/kreuzwerker/terraform-provider-docker/issues/10932))

* Add support for a list of pull_triggers within the docker_image resource. (#10845) ([#10845](https://github.com/kreuzwerker/terraform-provider-docker/issues/10845))

* Run `terraform fmt` on code examples (#12075) ([#12075](https://github.com/kreuzwerker/terraform-provider-docker/issues/12075))

* added support for linux capabilities (#12045) ([#12045](https://github.com/kreuzwerker/terraform-provider-docker/issues/12045) [#11623](https://github.com/kreuzwerker/terraform-provider-docker/issues/11623))

* Removing the note on docker provider about Terraform (#12676) ([#12676](https://github.com/kreuzwerker/terraform-provider-docker/issues/12676) [#12670](https://github.com/kreuzwerker/terraform-provider-docker/issues/12670))

* Massively add HCL source tag in docs Markdown files

* provider/docker network alias (#14710) ([#14710](https://github.com/kreuzwerker/terraform-provider-docker/issues/14710))

* Transfer docker provider website

* Merge branch 'master' of /Users/jake/terraform

* Transfer of provider website docs

* Transfer docker provider

* Merge branch 'master' of /home/ubuntu/terraform-vendor

* Transfer of provider code

* Updating Makefile + Add gitignore

* Update CHANGELOG.md

* v0.1.0

* Cleanup after v0.1.0 release

* Simplifying the GNUMakefile

* Ignore github.com/hashicorp/terraform/backend

* github.com/hashicorp/terraform/...@v0.10.0

* Merge pull request #18 from terraform-providers/vendor-tf-0.10 ([#18](https://github.com/kreuzwerker/terraform-provider-docker/issues/18))

* Correct git paths in the README.md file

* Correct comment in `docker_network` documentation

* Merge pull request #23 from JamesLaverack/patch-1 ([#23](https://github.com/kreuzwerker/terraform-provider-docker/issues/23))

* Make CoC and support channels more visible

* Update CHANGELOG.md

* v0.1.1

* Cleanup after v0.1.1 release

* add prefix 'library' only to official images in the path (#27) ([#27](https://github.com/kreuzwerker/terraform-provider-docker/issues/27))

* Update go-dockerclient

* Updated dockerclient to bf3bc17bb (#46) ([#46](https://github.com/kreuzwerker/terraform-provider-docker/issues/46))

* Updated private registry port to 15000 for fix teamcity build.

* Using absolute paths instead of relative in test scripts

* updated test script to clean up properly

* added 3rd test image and made build more generic

* Travis build uses docker version 18.03.0 to fix the build (#57) ([#57](https://github.com/kreuzwerker/terraform-provider-docker/issues/57))

* [ci skip] Updated CHANGELOG

* Add website + website-test targets

* Merge pull request #60 from terraform-providers/f-make-website ([#60](https://github.com/kreuzwerker/terraform-provider-docker/issues/60))

* correct provider organization & missing repository name

* [ci skip] update changelog

* Marks the links property as deprecated

* Merge pull request #47 from captn3m0/docker-link-warning ([#47](https://github.com/kreuzwerker/terraform-provider-docker/issues/47))

* Update CHANGELOG.md to include #47 ([#47](https://github.com/kreuzwerker/terraform-provider-docker/issues/47))

* Update/docker-18-03-1 (#68) ([#68](https://github.com/kreuzwerker/terraform-provider-docker/issues/68))

* Update CHANGELOG.md

* Update CHANGELOG.md

* update docker deps to 65bd038

* Migrate/docker-client (#70) ([#70](https://github.com/kreuzwerker/terraform-provider-docker/issues/70) [#32](https://github.com/kreuzwerker/terraform-provider-docker/issues/32))

* v1.0.0

* Cleanup after v1.0.0 release

* merge master

* merge vendor

* Merge branch 'master' of github.com:terraform-providers/terraform-provider-docker

* Update CHANGELOG.md

* v1.0.1

* Cleanup after v1.0.1 release

* v1.0.2

* Cleanup after v1.0.2 release

* [ci skip] updates CHANGELOG for windows support

* Image can be pulled with its repo digest (#97) ([#97](https://github.com/kreuzwerker/terraform-provider-docker/issues/97) [#79](https://github.com/kreuzwerker/terraform-provider-docker/issues/79))

* [ci skip] Fixes PR links in CHANGELOG

* v1.0.3

* Cleanup after v1.0.3 release

* Support for random external port for containers (#103) ([#103](https://github.com/kreuzwerker/terraform-provider-docker/issues/103) [#102](https://github.com/kreuzwerker/terraform-provider-docker/issues/102))

* v1.0.4

* Cleanup after v1.0.4 release

* [ci skip] updates CHANGELOG

* Container network fixes (#104) ([#104](https://github.com/kreuzwerker/terraform-provider-docker/issues/104) [#50](https://github.com/kreuzwerker/terraform-provider-docker/issues/50) [#9](https://github.com/kreuzwerker/terraform-provider-docker/issues/9) [#98](https://github.com/kreuzwerker/terraform-provider-docker/issues/98) [#107](https://github.com/kreuzwerker/terraform-provider-docker/issues/107))

* updates CHANGELOG

* Update doc

* bla

* Update doc

* Merges master

* cpuset added

* fmt

* Merge branch 'd-cpuset-added' of git://github.com/sunthera/terraform-provider-docker into sunthera/d-cpuset-added

* Merge branch 'sunthera/d-cpuset-added'

* Simplify the image options parser.

* Simplifies the image options parser. Closes #49. ([#49](https://github.com/kreuzwerker/terraform-provider-docker/issues/49))

* Merge branch 'mprihoda-wip/registry-parser'

* Merge branch 'bhuisgen-feature/docker-container-ipv6'

* Update tests

* Update doc

* Merge branch 'bhuisgen-feature/container-attach-logs'

* v1.1.0

* Cleanup after v1.1.0 release

* Updates network_advanced documentation for containers. Closes #109 ([#109](https://github.com/kreuzwerker/terraform-provider-docker/issues/109))

* [ci skip] updates CHANGELOG

* Update service.html fixed typo `tmpf_options` (#122) ([#122](https://github.com/kreuzwerker/terraform-provider-docker/issues/122))

* Updates changelog for docs corrections.

* Merge pull request #135 from terraform-providers/t-simplify-dockerfile ([#135](https://github.com/kreuzwerker/terraform-provider-docker/issues/135))

* updates CHANGELOG for v1.1.1

* v1.1.1

* Cleanup after v1.1.1 release

* provider: Ensured Go 1.11 in TravisCI and README

* deps: use go modules for dep mgmt

* deps: github.com/hashicorp/terraform@sdk-v0.11-with-go-modules

* remove turn modules off for gox installation

* Merge pull request #134 from terraform-providers/go-modules-2019-03-01 ([#134](https://github.com/kreuzwerker/terraform-provider-docker/issues/134))

* provider: Require Go 1.11 in TravisCI and README

* Upgrade to go 1.11

* Update to docker to 18.09 (#152) ([#152](https://github.com/kreuzwerker/terraform-provider-docker/issues/152) [#114](https://github.com/kreuzwerker/terraform-provider-docker/issues/114))

* updates changelog for v1.2.0

* updates docker api version to 1.39 and adapts changelog

* [ci skip]: Updates missing docs for changes of #147 ([#147](https://github.com/kreuzwerker/terraform-provider-docker/issues/147))

* v1.2.0

* Cleanup after v1.2.0 release

* Refactors test setup (#156) ([#156](https://github.com/kreuzwerker/terraform-provider-docker/issues/156))

* Update to Terraform 0.12 (#150). Closes #144 ([#150](https://github.com/kreuzwerker/terraform-provider-docker/issues/150) [#144](https://github.com/kreuzwerker/terraform-provider-docker/issues/144))

* updates CHANGELOG for v2.0.0

* v2.0.0

* Cleanup after v2.0.0 release

* Updates the docs (#165). Closes  #158 ([#165](https://github.com/kreuzwerker/terraform-provider-docker/issues/165) [#158](https://github.com/kreuzwerker/terraform-provider-docker/issues/158) [#147](https://github.com/kreuzwerker/terraform-provider-docker/issues/147) [#153](https://github.com/kreuzwerker/terraform-provider-docker/issues/153))

* [ci skip] updates CHANGELOG for v2.1.0

* v2.1.0

* Cleanup after v2.1.0 release

* v2.1.1

* Cleanup after v2.1.1 release

* Using APIVersion negotiation instead of fixed version (#177) ([#177](https://github.com/kreuzwerker/terraform-provider-docker/issues/177) [#137](https://github.com/kreuzwerker/terraform-provider-docker/issues/137))

* updates CHANGELOG for v2.2.0

* v2.2.0

* remove usage of config pkg (#183) ([#183](https://github.com/kreuzwerker/terraform-provider-docker/issues/183))

* [ci skip] updates CHANGELOG v2.3.0

* v2.3.0

* Cleanup after v2.3.0 release

* Remove duplicate start_period in docker_service documentation (#189) ([#189](https://github.com/kreuzwerker/terraform-provider-docker/issues/189))

* Calculate digest when missing from registry response headers (#188) ([#188](https://github.com/kreuzwerker/terraform-provider-docker/issues/188) [#186](https://github.com/kreuzwerker/terraform-provider-docker/issues/186))

* [ci skip] updates CHANGELOG v2.4.0

* v2.4.0

* Cleanup after v2.4.0 release

* Move to standalone plugin SDK (#200) ([#200](https://github.com/kreuzwerker/terraform-provider-docker/issues/200) [#197](https://github.com/kreuzwerker/terraform-provider-docker/issues/197))

* Updated logdrivers to match docker officially supported options (#207) ([#207](https://github.com/kreuzwerker/terraform-provider-docker/issues/207) [#204](https://github.com/kreuzwerker/terraform-provider-docker/issues/204))

* CI restructuring and migration to golang 1.13 (#202) ([#202](https://github.com/kreuzwerker/terraform-provider-docker/issues/202) [#198](https://github.com/kreuzwerker/terraform-provider-docker/issues/198))

* Remove logging drivers white-list (#209) ([#209](https://github.com/kreuzwerker/terraform-provider-docker/issues/209) [#207](https://github.com/kreuzwerker/terraform-provider-docker/issues/207) [#208](https://github.com/kreuzwerker/terraform-provider-docker/issues/208))

* [ci-skip] updates CHANGELOG for v2.5.0

* [ci skip] update CHANGELOG next release version

* v2.5.0

* Cleanup after v2.5.0 release

* Correct mounts block name (#218) ([#218](https://github.com/kreuzwerker/terraform-provider-docker/issues/218))

* [ci-skip] updates CHANGELOG for v2.6.0

* v2.6.0

* Cleanup after v2.6.0 release

* trying something different

* woops

* use %v not %s

* specify labels are optional

* try straight indices

* get a look at all the Attributes

* try this again?

* try this!

* skip the size var

* migrate the container resource

* secret and network cleanup

* volume

* service

* some more migration

* re-run travis, possible flaky test

* some debugging

* Merge branch 'xf/labels-iface' of git://github.com/xanderflood/terraform-provider-docker into xanderflood-xf/labels-iface

* Merge branch 'xanderflood-xf/labels-iface'

* allow zero replicas

* Merge branch 'xf/allow-zero-replicas' of git://github.com/xanderflood/terraform-provider-docker into xanderflood-xf/allow-zero-replicas

* Merge branch 'xanderflood-xf/allow-zero-replicas'

* Wait for state until endpoint spec is exposed

* Set TF_LOG=DEBUG

* Merge branch 'debug-issue-237' of https://github.com/suzuki-shunsuke/terraform-provider-docker into suzuki-shunsuke-debug-issue-237

* Merge branch 'suzuki-shunsuke-debug-issue-237'

* [ci-skip] updates CHANGELOG for v2.7.0

* v2.7.0

* Cleanup after v2.7.0 release

* [ci-skip] updates CHANGELOG for v2.7.1

* v2.7.1

* Cleanup after v2.7.1 release

* Set `Computed: true` and separate files of resourceDockerContainerV1 (#272) ([#272](https://github.com/kreuzwerker/terraform-provider-docker/issues/272))

* [ci-skip] updates CHANGELOG for v2.7.2

* v2.7.2

* Cleanup after v2.7.2 release

* Merge branch 'akomic-image_build' into master

* added docker_registry_image

* added documentation

* removed jsonmessage dependency

* removed assert dependency

* added error handling if context file not found

* Merge branch 'feature/resource-docker_registry_image' of git://github.com/edgarpoce/terraform-provider-docker into edgarpoce-feature/resource-docker_registry_image

* Merge branch 'edgarpoce-feature/resource-docker_registry_image' into master

* #250 no errors if auth config incomplete ([#250](https://github.com/kreuzwerker/terraform-provider-docker/issues/250))

* Merge branch 'bugfix/support_missing_auth_in_provider' of git://github.com/edgarpoce/terraform-provider-docker into edgarpoce-bugfix/support_missing_auth_in_provider

* Merge branch 'edgarpoce-bugfix/support_missing_auth_in_provider' into master

* Merge branch 'feat/ipv6-attrs' of git://github.com/ellsclytn/terraform-provider-docker into ellsclytn-feat/ipv6-attrs

* Merge branch 'ellsclytn-feat/ipv6-attrs' into master

* Merge branch 'master' of git://github.com/juliosueiras/terraform-provider-docker into juliosueiras-master

* Merge branch 'juliosueiras-master' into master

* Update service.html.markdown

* Merge branch 'patch-1' of git://github.com/motti24/terraform-provider-docker into motti24-patch-1

* Merge branch 'motti24-patch-1' into master


### Refactor

* refactor migration code and fix a couple tests


### Test

* Simplify Dockerfile(s)

* Skip test if swap limit isn't available (#136) ([#136](https://github.com/kreuzwerker/terraform-provider-docker/issues/136))

* tests for label migration


