package provider

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
	"strings"
	"time"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/fileutils"
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

	if buildOptions, ok := d.GetOk("build"); ok {
		buildOptionsMap := buildOptions.([]interface{})[0].(map[string]interface{})
		err := buildDockerRegistryImage(ctx, client, buildOptionsMap, pushOpts.FqName)
		if err != nil {
			return diag.Errorf("Error building docker image: %s", err)
		}
	}

	username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
	if err := pushDockerRegistryImage(ctx, client, pushOpts, username, password); err != nil {
		return diag.Errorf("Error pushing docker image: %s", err)
	}

	insecureSkipVerify := d.Get("insecure_skip_verify").(bool)
	digest, err := getImageDigestWithFallback(pushOpts, username, password, insecureSkipVerify)
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
	username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)

	insecureSkipVerify := d.Get("insecure_skip_verify").(bool)
	digest, err := getImageDigestWithFallback(pushOpts, username, password, insecureSkipVerify)
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
	username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
	digest := d.Get("sha256_digest").(string)
	err := deleteDockerRegistryImage(pushOpts, digest, username, password, true, false)
	if err != nil {
		err = deleteDockerRegistryImage(pushOpts, pushOpts.Tag, username, password, true, true)
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

func buildDockerRegistryImage(ctx context.Context, client *client.Client, buildOptions map[string]interface{}, fqName string) error {
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
	if lastIndex := strings.LastIndexByte(buildContext, ':'); (lastIndex > -1) && (buildContext[lastIndex+1] != filepath.Separator) {
		buildContext = buildContext[:lastIndex]
	}

	dockerContextTarPath, err := buildDockerImageContextTar(buildContext)
	if err != nil {
		return fmt.Errorf("unable to build context: %v", err)
	}
	defer os.Remove(dockerContextTarPath)
	dockerBuildContext, err := os.Open(dockerContextTarPath)
	if err != nil {
		return err
	}
	defer dockerBuildContext.Close()

	buildResponse, err := client.ImageBuild(ctx, dockerBuildContext, imageBuildOptions)
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
		return "", fmt.Errorf("cannot create temporary file - %v", err.Error())
	}

	defer tmpFile.Close()
	if _, err = os.Stat(buildContext); err != nil {
		return "", fmt.Errorf("unable to read build context - %v", err.Error())
	}

	excludes, err := build.ReadDockerignore(buildContext)
	if err != nil {
		return "", fmt.Errorf("unable to read .dockerignore file - %v", err.Error())
	}

	pm, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return "", fmt.Errorf("unable to create pattern matcher from .dockerignore excludes - %v", err.Error())
	}

	tw := tar.NewWriter(tmpFile)
	defer tw.Close()

	if err := filepath.Walk(buildContext, func(file string, info os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			return err
		}

		// if .dockerignore is present, ignore files from there
		rel, _ := filepath.Rel(buildContext, file)
		skip, err := pm.Matches(rel)
		if err != nil {
			return err
		}

		// adapted from https://github.com/moby/moby/blob/v20.10.7/pkg/archive/archive.go#L851
		if skip {
			log.Printf("[DEBUG] Skipping file/dir from image build '%v'", file)
			// If we want to skip this file and its a directory
			// then we should first check to see if there's an
			// excludes pattern (e.g. !dir/file) that starts with this
			// dir. If so then we can't skip this dir.

			// Its not a dir then so we can just return/skip.
			if !info.IsDir() {
				return nil
			}

			// No exceptions (!...) in patterns so just skip dir
			if !pm.Exclusions() {
				return filepath.SkipDir
			}

			dirSlash := file + string(filepath.Separator)

			for _, pat := range pm.Patterns() {
				if !pat.Exclusion() {
					continue
				}
				if strings.HasPrefix(pat.String()+string(filepath.Separator), dirSlash) {
					// found a match - so can't skip this dir
					return nil
				}
			}

			// No matching exclusion dir so just skip dir
			return filepath.SkipDir
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, buildContext, "", -1), string(filepath.Separator))

		// set archive metadata non deterministic
		header.Mode = 0
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""
		header.ModTime = time.Time{}
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}

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
	}); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func getDockerImageContextTarHash(dockerContextTarPath string) (string, error) {
	hasher := sha256.New()
	s, err := ioutil.ReadFile(dockerContextTarPath)
	if err != nil {
		return "", err
	}
	if _, err := hasher.Write(s); err != nil {
		return "", err
	}
	contextHash := hex.EncodeToString(hasher.Sum(nil))
	return contextHash, nil
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

func deleteDockerRegistryImage(pushOpts internalPushImageOptions, sha256Digest, username, password string, insecureSkipVerify, fallback bool) error {
	client := http.DefaultClient

	// DevSkim: ignore DS440000
	client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify}}

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
			if err != nil {
				return err
			}
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

func getImageDigestWithFallback(opts internalPushImageOptions, username, password string, insecureSkipVerify bool) (string, error) {
	digest, err := getImageDigest(opts.Registry, opts.Repository, opts.Tag, username, password, insecureSkipVerify, false)
	if err != nil {
		digest, err = getImageDigest(opts.Registry, opts.Repository, opts.Tag, username, password, insecureSkipVerify, true)
		if err != nil {
			return "", fmt.Errorf("unable to get digest: %s", err)
		}
	}
	return digest, nil
}

func createPushImageOptions(image string) internalPushImageOptions {
	pullOpts := parseImageOptions(image)
	if pullOpts.Registry == "" {
		pullOpts.Registry = "registry-1.docker.io"
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
