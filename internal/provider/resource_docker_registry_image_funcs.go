package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDockerRegistryImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient
	providerConfig := meta.(*ProviderConfig)
	name := d.Get("name").(string)
	log.Printf("[DEBUG] Creating docker image %s", name)

	pushOpts := createPushImageOptions(name)

	authConfig, err := getAuthConfigForRegistry(pushOpts.Registry, providerConfig)
	if err != nil {
		return diag.Errorf("resourceDockerRegistryImageCreate: Unable to get authConfig for registry: %s", err)
	}
	if err := pushDockerRegistryImage(ctx, client, pushOpts, authConfig.Username, authConfig.Password); err != nil {
		return diag.Errorf("Error pushing docker image: %s", err)
	}

	insecureSkipVerify := d.Get("insecure_skip_verify").(bool)
	digest, err := getImageDigestWithFallback(pushOpts, authConfig.ServerAddress, authConfig.Username, authConfig.Password, insecureSkipVerify)
	if err != nil {
		return diag.Errorf("Unable to create image, image not found: %s", err)
	}
	d.SetId(digest)
	d.Set("sha256_digest", digest)
	return nil
}

func resourceDockerRegistryImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(*ProviderConfig)
	name := d.Get("name").(string)
	pushOpts := createPushImageOptions(name)
	authConfig, err := getAuthConfigForRegistry(pushOpts.Registry, providerConfig)
	if err != nil {
		return diag.Errorf("resourceDockerRegistryImageRead: Unable to get authConfig for registry: %s", err)
	}

	insecureSkipVerify := d.Get("insecure_skip_verify").(bool)
	digest, err := getImageDigestWithFallback(pushOpts, authConfig.ServerAddress, authConfig.Username, authConfig.Password, insecureSkipVerify)
	if err != nil {
		log.Printf("Got error getting registry image digest: %s", err)
		d.SetId("")
		return nil
	}
	d.Set("sha256_digest", digest)
	return nil
}

func resourceDockerRegistryImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Get("keep_remotely").(bool) {
		return nil
	}
	providerConfig := meta.(*ProviderConfig)
	name := d.Get("name").(string)
	pushOpts := createPushImageOptions(name)
	authConfig, err := getAuthConfigForRegistry(pushOpts.Registry, providerConfig)
	if err != nil {
		return diag.Errorf("resourceDockerRegistryImageDelete: Unable to get authConfig for registry: %s", err)
	}

	digest := d.Get("sha256_digest").(string)
	err = deleteDockerRegistryImage(pushOpts, authConfig.ServerAddress, digest, authConfig.Username, authConfig.Password, true, false)
	if err != nil {
		err = deleteDockerRegistryImage(pushOpts, authConfig.ServerAddress, pushOpts.Tag, authConfig.Username, authConfig.Password, true, true)
		if err != nil {
			return diag.Errorf("Got error deleting registry image: %s", err)
		}
	}
	return nil
}

func resourceDockerRegistryImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceDockerRegistryImageRead(ctx, d, meta)
}

// Helpers
type internalPushImageOptions struct {
	Name               string
	FqName             string
	Registry           string
	NormalizedRegistry string
	Repository         string
	Tag                string
}

func createImageBuildOptions(buildOptions map[string]interface{}) types.ImageBuildOptions {
	mapOfInterfacesToMapOfStrings := func(mapOfInterfaces map[string]interface{}) map[string]string {
		mapOfStrings := make(map[string]string, len(mapOfInterfaces))
		for k, v := range mapOfInterfaces {
			mapOfStrings[k] = fmt.Sprintf("%v", v)
		}
		return mapOfStrings
	}

	interfaceArrayToStringArray := func(interfaceArray []interface{}) []string {
		stringArray := make([]string, len(interfaceArray))
		for i, v := range interfaceArray {
			stringArray[i] = fmt.Sprintf("%v", v)
		}
		return stringArray
	}

	mapToBuildArgs := func(buildArgsOptions map[string]interface{}) map[string]*string {
		buildArgs := make(map[string]*string, len(buildArgsOptions))
		for k, v := range buildArgsOptions {
			value := v.(string)
			buildArgs[k] = &value
		}
		return buildArgs
	}

	readULimits := func(options []interface{}) []*units.Ulimit {
		ulimits := make([]*units.Ulimit, len(options))
		for i, v := range options {
			ulimitOption := v.(map[string]interface{})
			ulimit := units.Ulimit{
				Name: ulimitOption["name"].(string),
				Hard: int64(ulimitOption["hard"].(int)),
				Soft: int64(ulimitOption["soft"].(int)),
			}
			ulimits[i] = &ulimit
		}
		return ulimits
	}

	readAuthConfigs := func(options []interface{}) map[string]types.AuthConfig {
		authConfigs := make(map[string]types.AuthConfig, len(options))
		for _, v := range options {
			authOptions := v.(map[string]interface{})
			auth := types.AuthConfig{
				Username:      authOptions["user_name"].(string),
				Password:      authOptions["password"].(string),
				Auth:          authOptions["auth"].(string),
				Email:         authOptions["email"].(string),
				ServerAddress: authOptions["server_address"].(string),
				IdentityToken: authOptions["identity_token"].(string),
				RegistryToken: authOptions["registry_token"].(string),
			}
			authConfigs[authOptions["host_name"].(string)] = auth
		}
		return authConfigs
	}

	buildImageOptions := types.ImageBuildOptions{}
	buildImageOptions.SuppressOutput = buildOptions["suppress_output"].(bool)
	buildImageOptions.RemoteContext = buildOptions["remote_context"].(string)
	buildImageOptions.NoCache = buildOptions["no_cache"].(bool)
	buildImageOptions.Remove = buildOptions["remove"].(bool)
	buildImageOptions.ForceRemove = buildOptions["force_remove"].(bool)
	buildImageOptions.PullParent = buildOptions["pull_parent"].(bool)
	buildImageOptions.Isolation = container.Isolation(buildOptions["isolation"].(string))
	buildImageOptions.CPUSetCPUs = buildOptions["cpu_set_cpus"].(string)
	buildImageOptions.CPUSetMems = buildOptions["cpu_set_mems"].(string)
	buildImageOptions.CPUShares = int64(buildOptions["cpu_shares"].(int))
	buildImageOptions.CPUQuota = int64(buildOptions["cpu_quota"].(int))
	buildImageOptions.CPUPeriod = int64(buildOptions["cpu_period"].(int))
	buildImageOptions.Memory = int64(buildOptions["memory"].(int))
	buildImageOptions.MemorySwap = int64(buildOptions["memory_swap"].(int))
	buildImageOptions.CgroupParent = buildOptions["cgroup_parent"].(string)
	buildImageOptions.NetworkMode = buildOptions["network_mode"].(string)
	buildImageOptions.ShmSize = int64(buildOptions["shm_size"].(int))
	buildImageOptions.Dockerfile = buildOptions["dockerfile"].(string)
	buildImageOptions.Ulimits = readULimits(buildOptions["ulimit"].([]interface{}))
	buildImageOptions.BuildArgs = mapToBuildArgs(buildOptions["build_args"].(map[string]interface{}))
	buildImageOptions.AuthConfigs = readAuthConfigs(buildOptions["auth_config"].([]interface{}))
	buildImageOptions.Labels = mapOfInterfacesToMapOfStrings(buildOptions["labels"].(map[string]interface{}))
	buildImageOptions.Squash = buildOptions["squash"].(bool)
	buildImageOptions.CacheFrom = interfaceArrayToStringArray(buildOptions["cache_from"].([]interface{}))
	buildImageOptions.SecurityOpt = interfaceArrayToStringArray(buildOptions["security_opt"].([]interface{}))
	buildImageOptions.ExtraHosts = interfaceArrayToStringArray(buildOptions["extra_hosts"].([]interface{}))
	buildImageOptions.Target = buildOptions["target"].(string)
	buildImageOptions.SessionID = buildOptions["session_id"].(string)
	buildImageOptions.Platform = buildOptions["platform"].(string)
	buildImageOptions.Version = types.BuilderVersion(buildOptions["version"].(string))
	buildImageOptions.BuildID = buildOptions["build_id"].(string)
	// outputs

	return buildImageOptions
}

func pushDockerRegistryImage(ctx context.Context, client *client.Client, pushOpts internalPushImageOptions, username string, password string) error {
	pushOptions := types.ImagePushOptions{}
	if username != "" {
		auth := types.AuthConfig{Username: username, Password: password}
		authBytes, err := json.Marshal(auth)
		if err != nil {
			return fmt.Errorf("Error creating push options: %s", err)
		}
		authBase64 := base64.URLEncoding.EncodeToString(authBytes)
		pushOptions.RegistryAuth = authBase64
	}

	out, err := client.ImagePush(ctx, pushOpts.FqName, pushOptions)
	if err != nil {
		return err
	}
	defer out.Close()

	type ErrorMessage struct {
		Error string
	}
	var errorMessage ErrorMessage
	buffIOReader := bufio.NewReader(out)
	for {
		streamBytes, err := buffIOReader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err := json.Unmarshal(streamBytes, &errorMessage); err != nil {
			return err
		}
		if errorMessage.Error != "" {
			return fmt.Errorf("Error pushing image: %s", errorMessage.Error)
		}
	}
	log.Printf("[DEBUG] Pushed image: %s", pushOpts.FqName)
	return nil
}

func getAuthConfigForRegistry(
	registryWithoutProtocol string,
	providerConfig *ProviderConfig) (types.AuthConfig, error) {
	if authConfig, ok := providerConfig.AuthConfigs.Configs[registryWithoutProtocol]; ok {
		return authConfig, nil
	}
	return types.AuthConfig{}, fmt.Errorf("no auth config found for registry %s in auth configs: %#v", registryWithoutProtocol, providerConfig.AuthConfigs.Configs)
}

func buildHttpClientForRegistry(registryAddressWithProtocol string, insecureSkipVerify bool) *http.Client {
	client := http.DefaultClient

	if strings.HasPrefix(registryAddressWithProtocol, "https://") {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify}, Proxy: http.ProxyFromEnvironment}
	} else {
		client.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}

	return client
}

func deleteDockerRegistryImage(pushOpts internalPushImageOptions, registryWithProtocol string, sha256Digest, username, password string, insecureSkipVerify, fallback bool) error {
	client := buildHttpClientForRegistry(registryWithProtocol, insecureSkipVerify)

	req, err := http.NewRequest("DELETE", registryWithProtocol+"/v2/"+pushOpts.Repository+"/manifests/"+sha256Digest, nil)
	if err != nil {
		return fmt.Errorf("Error deleting registry image: %s", err)
	}

	if username != "" {
		if pushOpts.Registry != "ghcr.io" && !isECRRepositoryURL(pushOpts.Registry) && !isAzureCRRepositoryURL(pushOpts.Registry) && pushOpts.Registry != "gcr.io" {
			req.SetBasicAuth(username, password)
		} else {
			if isECRRepositoryURL(pushOpts.Registry) {
				password = normalizeECRPasswordForHTTPUsage(password)
				req.Header.Add("Authorization", "Basic "+password)
			} else {
				req.Header.Add("Authorization", "Bearer "+base64.StdEncoding.EncodeToString([]byte(password)))
			}
		}
	}

	setupHTTPHeadersForRegistryRequests(req, fallback)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK, http.StatusAccepted, http.StatusNotFound:
		return nil

	// Either OAuth is required or the basic auth creds were invalid
	case http.StatusUnauthorized:
		if !strings.HasPrefix(resp.Header.Get("www-authenticate"), "Bearer") {
			return fmt.Errorf("Bad credentials: " + resp.Status)
		}

		token, err := getAuthToken(resp.Header.Get("www-authenticate"), username, password, client)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		oauthResp, err := client.Do(req)
		if err != nil {
			return err
		}
		switch oauthResp.StatusCode {
		case http.StatusOK, http.StatusAccepted, http.StatusNotFound:
			return nil
		default:
			return fmt.Errorf("Got bad response from registry: " + resp.Status)
		}
		// Some unexpected status was given, return an error
	default:
		return fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}

func getImageDigestWithFallback(opts internalPushImageOptions, serverAddress string, username, password string, insecureSkipVerify bool) (string, error) {
	digest, err := getImageDigest(opts.Registry, serverAddress, opts.Repository, opts.Tag, username, password, insecureSkipVerify, false)
	if err != nil {
		digest, err = getImageDigest(opts.Registry, serverAddress, opts.Repository, opts.Tag, username, password, insecureSkipVerify, true)
		if err != nil {
			return "", fmt.Errorf("unable to get digest: %s", err)
		}
	}
	return digest, nil
}

func createPushImageOptions(image string) internalPushImageOptions {
	pullOpts := parseImageOptions(image)
	pushOpts := internalPushImageOptions{
		Name:               image,
		Registry:           pullOpts.Registry,
		NormalizedRegistry: normalizeRegistryAddress(pullOpts.Registry),
		Repository:         pullOpts.Repository,
		Tag:                pullOpts.Tag,
		FqName:             fmt.Sprintf("%s/%s:%s", pullOpts.Registry, pullOpts.Repository, pullOpts.Tag),
	}
	return pushOpts
}
