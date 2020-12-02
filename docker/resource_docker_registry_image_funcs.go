package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

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

func buildDockerRegistryImage(client *client.Client, buildOptions map[string]interface{}, fqName string) error {
	type ErrorDetailMessage struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}

	type BuildImageResponseMessage struct {
		Error       string              `json:"error,omitempty"`
		ErrorDetail *ErrorDetailMessage `json:"errorDetail,omitempty"`
	}

	getError := func(body io.ReadCloser) error {
		dec := json.NewDecoder(body)
		for {
			message := BuildImageResponseMessage{}
			if err := dec.Decode(&message); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if message.ErrorDetail != nil {
				detail := message.ErrorDetail
				return fmt.Errorf("%v: %s", detail.Code, detail.Message)
			}
			if len(message.Error) > 0 {
				return fmt.Errorf("%s", message.Error)
			}
		}
		return nil
	}

	log.Printf("[DEBUG] Building docker image")
	imageBuildOptions := createImageBuildOptions(buildOptions)
	imageBuildOptions.Tags = []string{fqName}

	// the tar hash is passed only after the initial creation
	buildContext := buildOptions["context"].(string)
	if lastIndex := strings.LastIndexByte(buildContext, ':'); lastIndex > -1 {
		buildContext = buildContext[:lastIndex]
	}
	dockerContextTarPath, err := buildDockerImageContextTar(buildContext)
	if err != nil {
		return fmt.Errorf("Unable to build context %v", err)
	}
	defer os.Remove(dockerContextTarPath)
	dockerBuildContext, err := os.Open(dockerContextTarPath)
	defer dockerBuildContext.Close()

	buildResponse, err := client.ImageBuild(context.Background(), dockerBuildContext, imageBuildOptions)
	if err != nil {
		return err
	}
	defer buildResponse.Body.Close()

	err = getError(buildResponse.Body)
	if err != nil {
		return err
	}

	return nil
}

func buildDockerImageContextTar(buildContext string) (string, error) {
	// Create our Temp File:  This will create a filename like /tmp/terraform-provider-docker-123456.tar
	tmpFile, err := ioutil.TempFile(os.TempDir(), "terraform-provider-docker-*.tar")
	if err != nil {
		return "", fmt.Errorf("Cannot create temporary file - %v", err.Error())
	}

	defer tmpFile.Close()

	if _, err = os.Stat(buildContext); err != nil {
		return "", fmt.Errorf("Unable to read build context - %v", err.Error())
	}

	tw := tar.NewWriter(tmpFile)
	defer tw.Close()

	err = filepath.Walk(buildContext, func(file string, info os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			return err
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, buildContext, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !info.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})

	return tmpFile.Name(), nil
}

func getDockerImageContextTarHash(dockerContextTarPath string) (string, error) {
	hasher := sha256.New()
	s, err := ioutil.ReadFile(dockerContextTarPath)
	if err != nil {
		return "", err
	}
	hasher.Write(s)
	contextHash := hex.EncodeToString(hasher.Sum(nil))
	return contextHash, nil
}

func pushDockerRegistryImage(client *client.Client, pushOpts internalPushImageOptions, username string, password string) error {
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

	out, err := client.ImagePush(context.Background(), pushOpts.FqName, pushOptions)
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
		json.Unmarshal(streamBytes, &errorMessage)
		if errorMessage.Error != "" {
			return fmt.Errorf("Error pushing image: %s", errorMessage.Error)
		}
	}
	log.Printf("[DEBUG] Pushed image: %s", pushOpts.FqName)
	return nil
}

func getDockerRegistryImageRegistryUserNameAndPassword(
	pushOpts internalPushImageOptions,
	providerConfig *ProviderConfig) (string, string) {
	registry := pushOpts.NormalizedRegistry
	username := ""
	password := ""
	if authConfig, ok := providerConfig.AuthConfigs.Configs[registry]; ok {
		username = authConfig.Username
		password = authConfig.Password
	}
	return username, password
}

func deleteDockerRegistryImage(pushOpts internalPushImageOptions, sha256Digest, username, password string, fallback bool) error {
	client := http.DefaultClient

	// Allow insecure registries only for ACC tests
	// cuz we don't have a valid certs for this case
	if env, okEnv := os.LookupEnv("TF_ACC"); okEnv {
		if i, errConv := strconv.Atoi(env); errConv == nil && i >= 1 {
			cfg := &tls.Config{
				InsecureSkipVerify: true,
			}
			client.Transport = &http.Transport{
				TLSClientConfig: cfg,
			}
		}
	}

	req, err := http.NewRequest("DELETE", pushOpts.NormalizedRegistry+"/v2/"+pushOpts.Repository+"/manifests/"+sha256Digest, nil)
	if err != nil {
		return fmt.Errorf("Error deleting registry image: %s", err)
	}

	if username != "" {
		req.SetBasicAuth(username, password)
	}

	// We accept schema v2 manifests and manifest lists, and also OCI types
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	if fallback {
		// Fallback to this header if the registry does not support the v2 manifest like gcr.io
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v1+prettyjws")
	}

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
		if strings.HasPrefix(resp.Header.Get("www-authenticate"), "Bearer") {
			auth := parseAuthHeader(resp.Header.Get("www-authenticate"))
			params := url.Values{}
			params.Set("service", auth["service"])
			params.Set("scope", auth["scope"])
			tokenRequest, err := http.NewRequest("GET", auth["realm"]+"?"+params.Encode(), nil)
			if err != nil {
				return fmt.Errorf("Error creating registry request: %s", err)
			}

			if username != "" {
				tokenRequest.SetBasicAuth(username, password)
			}

			tokenResponse, err := client.Do(tokenRequest)
			if err != nil {
				return fmt.Errorf("Error during registry request: %s", err)
			}

			if tokenResponse.StatusCode != http.StatusOK {
				return fmt.Errorf("Got bad response from registry: " + tokenResponse.Status)
			}

			body, err := ioutil.ReadAll(tokenResponse.Body)
			if err != nil {
				return fmt.Errorf("Error reading response body: %s", err)
			}

			token := &TokenResponse{}
			err = json.Unmarshal(body, token)
			if err != nil {
				return fmt.Errorf("Error parsing OAuth token response: %s", err)
			}

			req.Header.Set("Authorization", "Bearer "+token.Token)
			oauthResp, err := client.Do(req)
			switch oauthResp.StatusCode {
			case http.StatusOK, http.StatusAccepted, http.StatusNotFound:
				return nil
			default:
				return fmt.Errorf("Got bad response from registry: " + resp.Status)
			}

		}

		return fmt.Errorf("Bad credentials: " + resp.Status)

		// Some unexpected status was given, return an error
	default:
		return fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}

func getImageDigestWithFallback(opts internalPushImageOptions, username, password string) (string, error) {
	digest, err := getImageDigest(opts.Registry, opts.Repository, opts.Tag, username, password, false)
	if err != nil {
		digest, err = getImageDigest(opts.Registry, opts.Repository, opts.Tag, username, password, true)
		if err != nil {
			return "", fmt.Errorf("Unable to get digest: %s", err)
		}
	}
	return digest, nil
}

func createPushImageOptions(image string) internalPushImageOptions {
	pullOpts := parseImageOptions(image)
	if pullOpts.Registry == "" {
		pullOpts.Registry = "registry.hub.docker.com"
	} else {
		pullOpts.Repository = strings.Replace(pullOpts.Repository, pullOpts.Registry+"/", "", 1)
	}
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

func resourceDockerRegistryImageCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient
	providerConfig := meta.(*ProviderConfig)
	name := d.Get("name").(string)
	log.Printf("[DEBUG] Creating docker image %s", name)

	pushOpts := createPushImageOptions(name)

	if buildOptions, ok := d.GetOk("build"); ok {
		buildOptionsMap := buildOptions.([]interface{})[0].(map[string]interface{})
		err := buildDockerRegistryImage(client, buildOptionsMap, pushOpts.FqName)
		if err != nil {
			return fmt.Errorf("Error building docker image: %s", err)
		}
	}

	username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
	if err := pushDockerRegistryImage(client, pushOpts, username, password); err != nil {
		return fmt.Errorf("Error pushing docker image: %s", err)
	}

	digest, err := getImageDigestWithFallback(pushOpts, username, password)
	if err != nil {
		return fmt.Errorf("Unable to create image, image not found: %s", err)
	}
	d.SetId(digest)
	d.Set("sha256_digest", digest)
	return nil
}

func resourceDockerRegistryImageRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(*ProviderConfig)
	name := d.Get("name").(string)
	pushOpts := createPushImageOptions(name)
	username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
	digest, err := getImageDigestWithFallback(pushOpts, username, password)
	if err != nil {
		log.Printf("Got error getting registry image digest: %s", err)
		d.SetId("")
		return nil
	}
	d.Set("sha256_digest", digest)
	return nil
}

func resourceDockerRegistryImageDelete(d *schema.ResourceData, meta interface{}) error {
	if d.Get("keep_remotely").(bool) {
		return nil
	}
	providerConfig := meta.(*ProviderConfig)
	name := d.Get("name").(string)
	pushOpts := createPushImageOptions(name)
	username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
	digest := d.Get("sha256_digest").(string)
	err := deleteDockerRegistryImage(pushOpts, digest, username, password, false)
	if err != nil {
		err = deleteDockerRegistryImage(pushOpts, pushOpts.Tag, username, password, true)
		if err != nil {
			return fmt.Errorf("Got error getting registry image digest: %s", err)
		}
	}
	return nil
}

func resourceDockerRegistryImageUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceDockerRegistryImageRead(d, meta)
}
