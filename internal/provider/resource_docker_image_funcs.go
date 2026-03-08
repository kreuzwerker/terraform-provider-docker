package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/command/image/build"
	dockerBuildTypes "github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/secrets/secretsprovider"
	"github.com/moby/go-archive"
	"github.com/pkg/errors"
)

func resourceDockerImageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	imageName := d.Get("name").(string)

	if value, ok := d.GetOk("build"); ok {
		for _, rawBuild := range value.(*schema.Set).List() {
			shouldReturn, d1 := buildImage(ctx, rawBuild, client, imageName)
			if shouldReturn {
				return d1
			}
		}
	}
	apiImage, err := findImage(ctx, imageName, client, meta.(*ProviderConfig).AuthConfigs, d.Get("platform").(string))
	if err != nil {
		return diag.Errorf("Unable to read Docker image into resource: %s", err)
	}

	d.SetId(apiImage.ID + d.Get("name").(string))
	return resourceDockerImageRead(ctx, d, meta)
}

func buildImage(ctx context.Context, rawBuild interface{}, client *client.Client, imageName string) (bool, diag.Diagnostics) {
	rawBuildValue := rawBuild.(map[string]interface{})
	useLegacyBuilder, _ := rawBuildValue["use_legacy_builder"].(bool)
	// now we need to determine whether we can use buildx or need to use the legacy builder
	canUseBuildx, err := canUseBuildx(ctx, client)
	if err != nil {
		return true, diag.FromErr(err)
	}
	if useLegacyBuilder {
		log.Printf("[DEBUG] use_legacy_builder=true, forcing legacy builder")
		canUseBuildx = false
	}

	log.Printf("[DEBUG] canUseBuildx: %v", canUseBuildx)
	// buildx is enabled
	if canUseBuildx {
		log.Printf("[DEBUG] Using buildx")
		dockerCli, err := createAndInitDockerCli(client)
		if err != nil {
			return true, diag.FromErr(fmt.Errorf("failed to create and init Docker CLI: %w", err))
		}

		options, err := mapBuildAttributesToBuildOptions(rawBuildValue, imageName, dockerCli)

		if err != nil {
			return true, diag.FromErr(fmt.Errorf("Error mapping build attributes: %v", err))
		}
		buildLogFile := rawBuildValue["build_log_file"].(string)

		log.Printf("[DEBUG] build options %#v", options)

		err = runBuild(ctx, dockerCli, options, buildLogFile)
		if err != nil {
			return true, diag.Errorf("Error running buildx build: %v", err)
		}
	} else {

		err := legacyBuildDockerImage(ctx, rawBuildValue, imageName, client)
		if err != nil {
			return true, diag.Errorf("Error running legacy build: %v", err)
		}
	}
	return false, nil
}

func resourceDockerImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	var data Data
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return diag.Errorf("Error reading docker image list: %s", err)
	}

	imageName := d.Get("name").(string)

	foundImage, err := searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return diag.Errorf("resourceDockerImageRead: error looking up local image %q: %s", imageName, err)
	}
	if foundImage == nil {
		log.Printf("[DEBUG] did not find image with name: %v", imageName)
		d.SetId("")
		return nil
	}

	repoDigest := determineRepoDigest(imageName, foundImage)

	// TODO mavogel: remove the appended name from the ID
	d.SetId(foundImage.ID + d.Get("name").(string))
	d.Set("image_id", foundImage.ID)
	d.Set("repo_digest", repoDigest)
	return nil
}

func resourceDockerImageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// We need to re-read in case switching parameters affects
	// the value of "latest" or others
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	imageName := d.Get("name").(string)
	_, err = findImage(ctx, imageName, client, meta.(*ProviderConfig).AuthConfigs, d.Get("platform").(string))
	if err != nil {
		return diag.Errorf("Unable to read Docker image into resource: %s", err)
	}

	return resourceDockerImageRead(ctx, d, meta)
}

func resourceDockerImageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*ProviderConfig).MakeClient(ctx, d)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Docker client: %w", err))
	}
	// TODO mavogel: add retries. see e.g. service updateFailsAndRollbackConvergeConfig test
	err = removeImage(ctx, d, client)
	if err != nil {
		return diag.Errorf("Unable to remove Docker image: %s", err)
	}
	d.SetId("")
	return nil
}

// Helpers
func searchLocalImages(ctx context.Context, client *client.Client, data Data, imageName string) (*image.Summary, error) {
	imageInspect, err := client.ImageInspect(ctx, imageName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to inspect image %s: %w", imageName, err)
	}

	_, err = json.MarshalIndent(imageInspect, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("error parsing inspect response: %w", err)
	}
	// log.Printf("[DEBUG] Docker image inspect from readFunc: %s", jsonObj)

	if apiImage, ok := data.DockerImages[imageInspect.ID]; ok {
		log.Printf("[DEBUG] found local image via imageName: %v", imageName)
		return apiImage, nil
	}
	return nil, nil
}

func removeImage(ctx context.Context, d *schema.ResourceData, client *client.Client) error {
	var data Data

	if keepLocally := d.Get("keep_locally").(bool); keepLocally {
		return nil
	}

	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return err
	}

	imageName := d.Get("name").(string)
	if imageName == "" {
		return fmt.Errorf("empty image name is not allowed")
	}

	foundImage, err := searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return fmt.Errorf("removeImage: error looking up local image %q: %w", imageName, err)
	}

	if foundImage != nil {
		imageDeleteResponseItems, err := client.ImageRemove(ctx, imageName, image.RemoveOptions{
			Force: d.Get("force_remove").(bool),
		})
		if err != nil {
			return err
		}
		indentedImageDeleteResponseItems, _ := json.MarshalIndent(imageDeleteResponseItems, "", "\t")
		log.Printf("[DEBUG] Deleted image items: \n%s", indentedImageDeleteResponseItems)
	}

	return nil
}

func fetchLocalImages(ctx context.Context, data *Data, client *client.Client) error {
	images, err := client.ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		return fmt.Errorf("unable to list Docker images: %w", err)
	}

	if data.DockerImages == nil {
		data.DockerImages = make(map[string]*image.Summary)
	}

	// Docker uses different nomenclatures in different places...sometimes a short
	// ID, sometimes long, etc. So we store both in the map so we can always find
	// the same image object. We store the tags and digests, too.
	for i, image := range images {
		data.DockerImages[image.ID[:12]] = &images[i]
		data.DockerImages[image.ID] = &images[i]
		for _, repotag := range image.RepoTags {
			data.DockerImages[repotag] = &images[i]
		}
		for _, repodigest := range image.RepoDigests {
			data.DockerImages[repodigest] = &images[i]
		}
	}

	return nil
}

func pullImage(ctx context.Context, data *Data, client *client.Client, authConfig *AuthConfigs, imageName string, platform string) error {
	pullOpts := parseImageOptions(imageName)

	auth := registry.AuthConfig{}
	if authConfig, ok := authConfig.Configs[pullOpts.Registry]; ok {
		auth = authConfig
	}

	encodedJSON, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("error creating auth config: %w", err)
	}

	out, err := client.ImagePull(ctx, imageName, image.PullOptions{
		RegistryAuth: base64.URLEncoding.EncodeToString(encodedJSON),
		Platform:     platform,
	})
	if err != nil {
		return fmt.Errorf("error pulling image %s: %w", imageName, err)
	}
	defer out.Close() //nolint:errcheck

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(out); err != nil {
		return err
	}
	s := buf.String()
	log.Printf("[DEBUG] pulled image %v: %v", imageName, s)

	return nil
}

type internalPullImageOptions struct {
	Repository string `qs:"fromImage"`
	Tag        string

	// Only required for Docker Engine 1.9 or 1.10 w/ Remote API < 1.21
	// and Docker Engine < 1.9
	// This parameter was removed in Docker Engine 1.11
	Registry string
}

// Parses an image name into a PullImageOptions struct.
// If the name has no registry, the registry-1.docker.io is used
// If the name has no tag, the tag "latest" is used
func parseImageOptions(image string) internalPullImageOptions {
	pullOpts := internalPullImageOptions{}

	// Pre-fill with image by default, update later if tag found
	pullOpts.Repository = image

	firstSlash := strings.Index(image, "/")

	// Detect the registry name - it should either contain port, be fully qualified or be localhost
	// If the image contains more than 2 path components, or at least one and the prefix looks like a hostname
	if strings.Count(image, "/") > 1 || firstSlash != -1 && (strings.ContainsAny(image[:firstSlash], ".:") || image[:firstSlash] == "localhost") {
		// registry/repo/image
		pullOpts.Registry = image[:firstSlash]
	}

	prefixLength := len(pullOpts.Registry)
	tagIndex := strings.Index(image[prefixLength:], ":")

	if tagIndex != -1 {
		// we have the tag, strip it
		pullOpts.Repository = image[:prefixLength+tagIndex]
		pullOpts.Tag = image[prefixLength+tagIndex+1:]
		digestIndex := strings.Index(pullOpts.Tag, "@")
		if digestIndex != -1 {
			log.Printf("[INFO] Found digest in tag: %s, we are using the digest for pulling the image from the registry", pullOpts.Tag)
			// prefer pinned digest over tag name
			pullOpts.Tag = pullOpts.Tag[digestIndex+1:]
		}
	}

	if pullOpts.Tag == "" {
		pullOpts.Tag = "latest"
	}

	// Use the official Docker Hub if a registry isn't specified
	if pullOpts.Registry == "" {
		pullOpts.Registry = "registry-1.docker.io"
		// Docker prefixes 'library' to official images in the path; 'consul' becomes 'library/consul'
		if !strings.Contains(pullOpts.Repository, "/") {
			pullOpts.Repository = "library/" + pullOpts.Repository
		}
	} else {
		// Otherwise, filter the registry name out of the repo name
		pullOpts.Repository = strings.Replace(pullOpts.Repository, pullOpts.Registry+"/", "", 1)
	}

	return pullOpts
}

func findImage(ctx context.Context, imageName string, client *client.Client, authConfig *AuthConfigs, platform string) (*image.Summary, error) {
	if imageName == "" {
		return nil, fmt.Errorf("empty image name is not allowed")
	}

	var data Data
	// load local images into the data structure
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return nil, err
	}
	foundImage, err := searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return nil, fmt.Errorf("findImage1: error looking up local image %q: %w", imageName, err)
	}
	if foundImage != nil {
		return foundImage, nil
	}
	if err := pullImage(ctx, &data, client, authConfig, imageName, platform); err != nil {
		return nil, fmt.Errorf("unable to pull image %s: %s", imageName, err)
	}

	// update the data structure of the images
	if err := fetchLocalImages(ctx, &data, client); err != nil {
		return nil, err
	}

	foundImage, err = searchLocalImages(ctx, client, data, imageName)
	if err != nil {
		return nil, fmt.Errorf("findImage2: error looking up local image %q: %w", imageName, err)
	}
	if foundImage != nil {
		return foundImage, nil
	}

	return nil, fmt.Errorf("unable to find or pull image %s", imageName)
}

func legacyBuildDockerImage(ctx context.Context, rawBuild map[string]interface{}, imageName string, client *client.Client) error {
	var (
		err error
	)

	log.Printf("[DEBUG] Building docker image")
	buildOptions := createImageBuildOptions(rawBuild)

	tags := []string{imageName}
	for _, t := range rawBuild["tag"].([]interface{}) {
		tags = append(tags, t.(string))
	}
	buildOptions.Tags = tags

	buildContext := rawBuild["context"].(string)

	// Each build must have its own session. Never reuse buildKitSession!
	buildKitSession, sessionDone := enableBuildKitIfSupported(ctx, client, &buildOptions)
	// If Buildkit is enabled, try to parse and use secrets if present.
	if buildKitSession != nil {
		if secretsRaw, secretsDefined := rawBuild["secrets"]; secretsDefined {
			parsedSecrets := parseBuildSecrets(secretsRaw)

			store, err := secretsprovider.NewStore(parsedSecrets)
			if err != nil {
				return err
			}

			provider := secretsprovider.NewSecretProvider(store)
			buildKitSession.Allow(provider)
		}
	}

	buildCtx, relDockerfile, err := prepareBuildContext(buildContext, buildOptions.Dockerfile)
	if err != nil {
		if buildKitSession != nil {
			log.Printf("[DEBUG] Closing BuildKit session (first error path): ID=%s", buildKitSession.ID())
			buildKitSession.Close() //nolint:errcheck
			if sessionDone != nil {
				<-sessionDone
			}
		}
		return err
	}
	buildOptions.Dockerfile = relDockerfile

	var response dockerBuildTypes.ImageBuildResponse
	response, err = client.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		if buildKitSession != nil {
			log.Printf("[DEBUG] Closing BuildKit session (second error path): ID=%s", buildKitSession.ID())
			buildKitSession.Close() //nolint:errcheck
			if sessionDone != nil {
				<-sessionDone
			}
		}
		return err
	}
	defer response.Body.Close() //nolint:errcheck

	buildResult, err := decodeBuildMessages(response)
	if err != nil {
		return fmt.Errorf("%s\n\n%s", err, buildResult)
	}
	if buildKitSession != nil {
		log.Printf("[DEBUG] Closing BuildKit session (success path): ID=%s", buildKitSession.ID())
		buildKitSession.Close() //nolint:errcheck
		if sessionDone != nil {
			<-sessionDone
		}
	}
	return nil
}

const minBuildkitDockerVersion = "1.39"

func enableBuildKitIfSupported(
	ctx context.Context,
	client *client.Client,
	buildOptions *dockerBuildTypes.ImageBuildOptions,
) (*session.Session, chan struct{}) {
	dockerClientVersion := client.ClientVersion()
	log.Printf("[DEBUG] DockerClientVersion: %v, minBuildKitDockerVersion: %v\n", dockerClientVersion, minBuildkitDockerVersion)
	if versions.GreaterThanOrEqualTo(dockerClientVersion, minBuildkitDockerVersion) {
		// Generate a unique session key for each build
		sessionKey := fmt.Sprintf("docker-provider-%d", rand.Int63())
		log.Printf("[DEBUG] Creating BuildKit session with key: %s", sessionKey)
		s, _ := session.NewSession(ctx, sessionKey)
		dialSession := func(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
			return client.DialHijack(ctx, "/session", proto, meta)
		}
		done := make(chan struct{})
		go func() {
			s.Run(ctx, dialSession) //nolint:errcheck
			close(done)
		}()
		buildOptions.SessionID = s.ID()
		buildOptions.Version = dockerBuildTypes.BuilderBuildKit
		return s, done
	} else {
		buildOptions.Version = dockerBuildTypes.BuilderV1
		return nil, nil
	}
}

// resolveDockerfilePath resolves the dockerfile path relative to the context directory.
// It handles both absolute and relative paths consistently and determines if the dockerfile
// is inside or outside the build context.
// Returns:
// - contextDir: the absolute path to the build context
// - dockerfilePath: the path to the dockerfile (absolute if outside context, relative if inside)
// - isOutsideContext: true if dockerfile is outside the context directory
func resolveDockerfilePath(specifiedContext string, specifiedDockerfile string) (contextDir string, dockerfilePath string, isOutsideContext bool, err error) {
	// Expand and make context path absolute
	contextDir, err = homedir.Expand(specifiedContext)
	if err != nil {
		return "", "", false, fmt.Errorf("error expanding context path: %w", err)
	}

	contextDir, err = filepath.Abs(contextDir)
	if err != nil {
		return "", "", false, fmt.Errorf("error getting absolute context path: %w", err)
	}

	log.Printf("[DEBUG] Resolved context directory: %s", contextDir)

	// Handle dockerfile path
	var absDockerfilePath string
	if filepath.IsAbs(specifiedDockerfile) {
		// Dockerfile path is already absolute
		absDockerfilePath = specifiedDockerfile
	} else {
		// Dockerfile path is relative - resolve it relative to the context
		absDockerfilePath = filepath.Join(contextDir, specifiedDockerfile)
	}

	// Clean the path
	absDockerfilePath = filepath.Clean(absDockerfilePath)

	// Check if dockerfile exists
	if _, err := os.Stat(absDockerfilePath); err != nil {
		if os.IsNotExist(err) {
			return "", "", false, fmt.Errorf("dockerfile not found at path: %s", absDockerfilePath)
		}
		return "", "", false, fmt.Errorf("error accessing dockerfile at %s: %w", absDockerfilePath, err)
	}

	// Determine if the dockerfile is inside or outside the context
	relPath, err := filepath.Rel(contextDir, absDockerfilePath)
	if err != nil {
		return "", "", false, fmt.Errorf("error computing relative path: %w", err)
	}

	// If the relative path starts with "..", the dockerfile is outside the context
	if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
		isOutsideContext = true
	} else {
		// Dockerfile is inside the context - use relative path
		isOutsideContext = false
	}

	log.Printf("[DEBUG] Resolved dockerfile path: context=%s, dockerfile=%s, isOutside=%v", contextDir, dockerfilePath, isOutsideContext)
	return contextDir, absDockerfilePath, isOutsideContext, nil
}

func prepareBuildContext(specifiedContext string, specifiedDockerfile string) (io.ReadCloser, string, error) {
	var (
		dockerfileCtx io.ReadCloser
		contextDir    string
		relDockerfile string
		err           error
	)

	contextDir, relDockerfile, err = build.GetContextFromLocalDir(specifiedContext, specifiedDockerfile)

	log.Printf("[DEBUG] contextDir %s", contextDir)
	log.Printf("[DEBUG] relDockerfile %s", relDockerfile)
	if err == nil && strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		// Dockerfile is outside of build-context; read the Dockerfile and pass it as dockerfileCtx
		log.Printf("[DEBUG] Dockerfile is outside of build-context")
		dockerfileCtx, err = os.Open(specifiedDockerfile)
		if err != nil {
			return nil, "", errors.Errorf("unable to open Dockerfile: %v", err)
		}
		defer dockerfileCtx.Close() //nolint:errcheck
	}
	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return nil, "", err
	}

	// specifiedDockerfile = archive.CanonicalTarNameForPath(specifiedDockerfile)
	excludes = build.TrimBuildFilesFromExcludes(excludes, specifiedDockerfile, false)
	log.Printf("[DEBUG] Excludes: %v", excludes)
	buildCtx := getBuildContext(contextDir, excludes)

	// replace Dockerfile if it was added from stdin or a file outside the build-context, and there is archive context
	if dockerfileCtx != nil && buildCtx != nil {
		log.Printf("[DEBUG] Adding dockerfile to build context")
		buildCtx, relDockerfile, err = build.AddDockerfileToBuildContext(dockerfileCtx, buildCtx)
		if err != nil {
			return nil, "", err
		}
	}

	// Compress build context to avoid Docker misinterpreting it as plain text
	if buildCtx != nil {
		buildCtx, err = build.Compress(buildCtx)
		if err != nil {
			return nil, "", err
		}
	}

	if relDockerfile != "" {
		return buildCtx, relDockerfile, nil
	}
	return buildCtx, specifiedDockerfile, nil
}

func getBuildContext(filePath string, excludes []string) io.ReadCloser {
	filePath, _ = homedir.Expand(filePath)
	//TarWithOptions works only with absolute paths in Windows.
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("Invalid build directory: %s", filePath)
	}
	ctx, _ := archive.TarWithOptions(filePath, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &archive.ChownOpts{UID: 0, GID: 0},
	})
	return ctx
}

func decodeBuildMessages(response dockerBuildTypes.ImageBuildResponse) (string, error) {
	buf := new(bytes.Buffer)
	buildErr := error(nil)

	dec := json.NewDecoder(response.Body)
	for dec.More() {
		var m jsonmessage.JSONMessage
		err := dec.Decode(&m)
		if err != nil {
			return buf.String(), fmt.Errorf("problem decoding message from docker daemon: %s", err)
		}

		if err := m.Display(buf, false); err != nil {
			return "", err
		}

		if m.Error != nil {
			buildErr = fmt.Errorf("unable to build image")
		}
	}
	log.Printf("[DEBUG] %s", buf.String())

	return buf.String(), buildErr
}

func parseBuildSecrets(secretsRaw interface{}) []secretsprovider.Source {
	options := secretsRaw.([]interface{})

	secrets := make([]secretsprovider.Source, len(options))
	for i, option := range options {
		secretRaw := option.(map[string]interface{})
		source := secretsprovider.Source{
			ID:       secretRaw["id"].(string),
			FilePath: secretRaw["src"].(string),
			Env:      secretRaw["env"].(string),
		}
		secrets[i] = source
	}

	return secrets
}
