package provider

// taken from https://github.com/docker/buildx/blob/master/commands/build.go and heavily modified
// to fit the needs of the provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/containerd/console"
	"github.com/docker/buildx/build"
	"github.com/docker/buildx/builder"
	"github.com/docker/buildx/controller"
	cbuild "github.com/docker/buildx/controller/build"
	"github.com/docker/buildx/controller/control"
	controllererrors "github.com/docker/buildx/controller/errdefs"
	controllerapi "github.com/docker/buildx/controller/pb"
	"github.com/docker/buildx/monitor"
	"github.com/docker/buildx/util/buildflags"
	"github.com/docker/buildx/util/confutil"
	"github.com/docker/buildx/util/desktop"
	"github.com/docker/buildx/util/ioset"
	"github.com/docker/buildx/util/progress"
	"github.com/docker/cli/cli/command"
	dockeropts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/versions"
	dockerclient "github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/subrequests"
	"github.com/moby/buildkit/frontend/subrequests/lint"
	"github.com/moby/buildkit/frontend/subrequests/outline"
	"github.com/moby/buildkit/frontend/subrequests/targets"
	"github.com/moby/buildkit/solver/errdefs"
	solverpb "github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/moby/sys/atomicwriter"
	"github.com/morikuni/aec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	// import drivers otherwise factories are empty
	// for --driver output flag usage
	_ "github.com/docker/buildx/driver/docker"
	_ "github.com/docker/buildx/driver/docker-container"
	_ "github.com/docker/buildx/driver/kubernetes"
	_ "github.com/docker/buildx/driver/remote"
)

type buildOptions struct {
	allow          []string
	annotations    []string
	buildArgs      []string
	cacheFrom      []string
	cacheTo        []string
	cgroupParent   string
	contextPath    string
	contexts       []string
	dockerfileName string
	extraHosts     []string
	imageIDFile    string
	labels         []string
	networkMode    string
	noCacheFilter  []string
	outputs        []string
	platforms      []string
	callFunc       string
	secrets        []string
	shmSize        dockeropts.MemBytes
	ssh            []string
	tags           []string
	target         string
	ulimits        *dockeropts.UlimitOpt

	attests    []string
	sbom       string
	provenance string

	quiet bool

	builder      string
	metadataFile string
	noCache      bool
	pull         bool
	exportPush   bool
	exportLoad   bool

	control.ControlOptions

	invokeConfig *invokeConfig
}

func (o *buildOptions) toControllerOptions() (*controllerapi.BuildOptions, error) {
	var err error

	buildArgs, err := listToMap(o.buildArgs, true)
	if err != nil {
		return nil, err
	}

	labels, err := listToMap(o.labels, false)
	if err != nil {
		return nil, err
	}

	opts := controllerapi.BuildOptions{
		Allow:          o.allow,
		Annotations:    o.annotations,
		BuildArgs:      buildArgs,
		CgroupParent:   o.cgroupParent,
		ContextPath:    o.contextPath,
		DockerfileName: o.dockerfileName,
		ExtraHosts:     o.extraHosts,
		Labels:         labels,
		NetworkMode:    o.networkMode,
		NoCacheFilter:  o.noCacheFilter,
		Platforms:      o.platforms,
		ShmSize:        int64(o.shmSize),
		Tags:           o.tags,
		Target:         o.target,
		Ulimits:        dockerUlimitToControllerUlimit(o.ulimits),
		Builder:        o.builder,
		NoCache:        o.noCache,
		Pull:           o.pull,
		ExportPush:     o.exportPush,
		ExportLoad:     o.exportLoad,
	}

	// TODO: extract env var parsing to a method easily usable by library consumers
	if v := os.Getenv("SOURCE_DATE_EPOCH"); v != "" {
		if _, ok := opts.BuildArgs["SOURCE_DATE_EPOCH"]; !ok {
			opts.BuildArgs["SOURCE_DATE_EPOCH"] = v
		}
	}

	opts.SourcePolicy, err = build.ReadSourcePolicy()
	if err != nil {
		return nil, err
	}

	inAttests := slices.Clone(o.attests)
	if o.provenance != "" {
		inAttests = append(inAttests, buildflags.CanonicalizeAttest("provenance", o.provenance))
	}
	if o.sbom != "" {
		inAttests = append(inAttests, buildflags.CanonicalizeAttest("sbom", o.sbom))
	}
	opts.Attests, err = buildflags.ParseAttests(inAttests)
	if err != nil {
		return nil, err
	}

	opts.NamedContexts, err = buildflags.ParseContextNames(o.contexts)
	if err != nil {
		return nil, err
	}

	opts.Exports, err = buildflags.ParseExports(o.outputs)
	if err != nil {
		return nil, err
	}
	for _, e := range opts.Exports {
		if (e.Type == client.ExporterLocal || e.Type == client.ExporterTar) && o.imageIDFile != "" {
			return nil, errors.Errorf("local and tar exporters are incompatible with image ID file")
		}
	}

	cacheFrom, err := buildflags.ParseCacheEntry(o.cacheFrom)
	if err != nil {
		return nil, err
	}
	opts.CacheFrom = cacheFrom.ToPB()

	cacheTo, err := buildflags.ParseCacheEntry(o.cacheTo)
	if err != nil {
		return nil, err
	}
	opts.CacheTo = cacheTo.ToPB()

	opts.Secrets, err = buildflags.ParseSecretSpecs(o.secrets)
	if err != nil {
		return nil, err
	}
	opts.SSH, err = buildflags.ParseSSHSpecs(o.ssh)
	if err != nil {
		return nil, err
	}

	opts.CallFunc, err = buildflags.ParseCallFunc(o.callFunc)
	if err != nil {
		return nil, err
	}

	prm := confutil.MetadataProvenance()
	if opts.CallFunc != nil || len(o.metadataFile) == 0 {
		prm = confutil.MetadataProvenanceModeDisabled
	}
	opts.ProvenanceResponseMode = string(prm)

	return &opts, nil
}

func mapBuildAttributesToBuildOptions(buildAttributes map[string]interface{}, imageName string) (buildOptions, error) {
	options := buildOptions{}
	if dockerfile, ok := buildAttributes["dockerfile"].(string); ok {
		options.dockerfileName = dockerfile
	}

	options.contextPath = buildAttributes["context"].(string)
	options.exportLoad = true

	options.tags = append(options.tags, imageName)
	for _, t := range buildAttributes["tag"].([]interface{}) {
		options.tags = append(options.tags, t.(string))
	}

	if builder, ok := buildAttributes["builder"].(string); ok {
		options.builder = builder
	}

	if remove, ok := buildAttributes["remove"].(bool); ok {
		options.noCache = !remove
	}

	if secrets, ok := buildAttributes["secrets"].([]interface{}); ok {
		for _, secret := range secrets {
			if secretMap, ok := secret.(map[string]interface{}); ok {
				// Construct the secret string in the format [type=env,]id=<ID>[,env=<VARIABLE>]
				secretStr := ""

				id, _ := secretMap["id"].(string)
				if env, ok := secretMap["env"].(string); ok && env != "" {
					secretStr += fmt.Sprintf("type=env,id=%s,env=%s", id, env)
				}

				if src, ok := secretMap["src"].(string); ok && src != "" {
					secretStr += fmt.Sprintf("type=file,id=%s,src=%s", id, src)
				}

				options.secrets = append(options.secrets, secretStr)
			}
		}
	}

	if labels, ok := buildAttributes["label"].(map[string]interface{}); ok {
		for key, value := range labels {
			if valueStr, ok := value.(string); ok {
				options.labels = append(options.labels, fmt.Sprintf("%s=%s", key, valueStr))
			}
		}
	}

	if suppressOutput, ok := buildAttributes["suppress_output"].(bool); ok {
		options.quiet = suppressOutput
	}

	if noCache, ok := buildAttributes["no_cache"].(bool); ok {
		options.noCache = noCache
	}

	if pullParent, ok := buildAttributes["pull_parent"].(bool); ok {
		options.pull = pullParent
	}

	if isolation, ok := buildAttributes["isolation"].(string); ok {
		options.networkMode = isolation
	}

	if cpuSetCpus, ok := buildAttributes["cpu_set_cpus"].(string); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("cpusetcpus=%s", cpuSetCpus))
	}

	if cpuSetMems, ok := buildAttributes["cpu_set_mems"].(string); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("cpusetmems=%s", cpuSetMems))
	}

	if cpuShares, ok := buildAttributes["cpu_shares"].(int); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("cpushares=%d", cpuShares))
	}

	if cpuQuota, ok := buildAttributes["cpu_quota"].(int); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("cpuquota=%d", cpuQuota))
	}

	if cpuPeriod, ok := buildAttributes["cpu_period"].(int); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("cpuperiod=%d", cpuPeriod))
	}

	if memory, ok := buildAttributes["memory"].(int); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("memory=%d", memory))
	}

	if memorySwap, ok := buildAttributes["memory_swap"].(int); ok {
		options.buildArgs = append(options.buildArgs, fmt.Sprintf("memoryswap=%d", memorySwap))
	}

	if cgroupParent, ok := buildAttributes["cgroup_parent"].(string); ok {
		options.cgroupParent = cgroupParent
	}

	if networkMode, ok := buildAttributes["network_mode"].(string); ok {
		options.networkMode = networkMode
	}

	if shmSize, ok := buildAttributes["shm_size"].(int); ok {
		options.shmSize = dockeropts.MemBytes(shmSize)
	}

	options.ulimits = dockeropts.NewUlimitOpt(nil)
	if ulimits, ok := buildAttributes["ulimit"].([]interface{}); ok {
		ulimitOpt := &dockeropts.UlimitOpt{}
		for _, ulimit := range ulimits {
			if ulimitMap, ok := ulimit.(map[string]interface{}); ok {
				name, _ := ulimitMap["name"].(string)
				hard, _ := ulimitMap["hard"].(int)
				soft, _ := ulimitMap["soft"].(int)
				ulimitOpt.Set(fmt.Sprintf("%s=%d:%d", name, soft, hard)) // nolint:errcheck
			}
		}
		options.ulimits = ulimitOpt
	}

	if buildArgs, ok := buildAttributes["build_args"].(map[string]interface{}); ok {
		for key, value := range buildArgs {
			if valueStr, ok := value.(string); ok {
				options.buildArgs = append(options.buildArgs, fmt.Sprintf("%s=%s", key, valueStr))
			}
		}
	}

	if extraHosts, ok := buildAttributes["extra_hosts"].([]interface{}); ok {
		for _, host := range extraHosts {
			if hostStr, ok := host.(string); ok {
				options.extraHosts = append(options.extraHosts, hostStr)
			}
		}
	}

	if target, ok := buildAttributes["target"].(string); ok {
		options.target = target
	}

	if platform, ok := buildAttributes["platform"].(string); ok {
		options.platforms = append(options.platforms, platform)
	}

	return options, nil
}

func canUseBuildx(ctx context.Context, client *dockerclient.Client) (bool, error) {
	var buildKitDisabled, useBuilder bool

	// check DOCKER_BUILDKIT env var is not empty
	// if it is assume we want to use the builder component
	if v := os.Getenv("DOCKER_BUILDKIT"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("DOCKER_BUILDKIT environment variable expects boolean value: %w", err)
		}
		if !enabled {
			buildKitDisabled = true
		} else {
			useBuilder = true
		}
	}
	si, _ := client.Ping(ctx)

	if !useBuilder {
		if si.BuilderVersion != types.BuilderBuildKit && si.OSType == "windows" {
			// The daemon didn't advertise BuildKit as the preferred builder,
			// so use the legacy builder, which is still the default for
			// Windows / WCOW.
			return false, nil
		}
	}

	if buildKitDisabled {
		// When using a Linux daemon, print a warning that the legacy builder
		// is deprecated. For Windows / WCOW, BuildKit is still experimental,
		// so we don't print this warning, even if the daemon advertised that
		// it supports BuildKit.
		if si.OSType != "windows" {
			tflog.Warn(ctx, `DEPRECATED: The legacy builder is deprecated and will be removed in a future release.
            BuildKit is currently disabled; enable it by removing the DOCKER_BUILDKIT=0
            environment-variable.`)
		}
		return false, nil
	}

	return true, nil
}

func runBuild(ctx context.Context, dockerCli command.Cli, options buildOptions, buildLogFile string) (err error) {

	if buildLogFile == "" {
		buildLogFile = os.DevNull
	}

	logFile, err := os.OpenFile(buildLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open build log file: %w", err)
	}
	defer logFile.Close() // nolint:errcheck

	logger := log.New(logFile, "", log.LstdFlags)

	opts, err := options.toControllerOptions()
	if err != nil {
		return err
	}

	// Avoid leaving a stale file if we eventually fail
	if options.imageIDFile != "" {
		if err := os.Remove(options.imageIDFile); err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "removing image ID file")
		}
	}

	contextPathHash := options.contextPath
	if absContextPath, err := filepath.Abs(contextPathHash); err == nil {
		contextPathHash = absContextPath
	}
	b, err := builder.New(dockerCli,
		builder.WithName(options.builder),
		builder.WithContextPathHash(contextPathHash),
	)
	if err != nil {
		return err
	}
	_, err = b.LoadNodes(ctx)
	if err != nil {
		return err
	}

	var term bool
	if _, err := console.ConsoleFromFile(logFile); err == nil {
		term = true
	}

	ctx2, cancel := context.WithCancelCause(context.TODO())
	defer func() { cancel(errors.WithStack(context.Canceled)) }()
	progressMode := progressui.PlainMode
	if err != nil {
		return err
	}
	var printer *progress.Printer
	printer, err = progress.NewPrinter(ctx2, logFile, progressMode,
		progress.WithDesc(
			fmt.Sprintf("building with %q instance using %s driver", b.Name, b.Driver),
			fmt.Sprintf("%s:%s", b.Driver, b.Name),
		),
		progress.WithOnClose(func() {
			printWarnings(logFile, printer.Warnings(), progressMode)
		}),
	)
	if err != nil {
		logger.Printf("error creating progress printer: %v", err)
		return err
	}

	var resp *client.SolveResponse
	var inputs *build.Inputs
	var retErr error
	if confutil.IsExperimental() {
		resp, inputs, retErr = runControllerBuild(ctx, dockerCli, opts, options, printer)
	} else {
		resp, inputs, retErr = runBasicBuild(ctx, dockerCli, opts, printer)
	}

	if err := printer.Wait(); retErr == nil {
		retErr = err
	}

	if retErr != nil {
		return retErr
	}

	desktop.PrintBuildDetails(logFile, printer.BuildRefs(), term)

	if options.imageIDFile != "" {
		if err := os.WriteFile(options.imageIDFile, []byte(getImageID(resp.ExporterResponse)), 0644); err != nil {
			return errors.Wrap(err, "writing image ID file")
		}
	}
	if options.metadataFile != "" {
		dt := decodeExporterResponse(resp.ExporterResponse)
		if opts.CallFunc == nil {
			if warnings := printer.Warnings(); len(warnings) > 0 && confutil.MetadataWarningsEnabled() {
				dt["buildx.build.warnings"] = warnings
			}
		}
		if err := writeMetadataFile(options.metadataFile, dt); err != nil {
			return err
		}
	}
	if opts.CallFunc != nil {
		if exitcode, err := printResult(dockerCli.Out(), opts.CallFunc, resp.ExporterResponse, options.target, inputs); err != nil {
			return err
		} else if exitcode != 0 {
			os.Exit(exitcode)
		}
	}
	if v, ok := resp.ExporterResponse["frontend.result.inlinemessage"]; ok {
		fmt.Fprintf(dockerCli.Out(), "\n%s\n", v) // nolint:errcheck
		return nil
	}
	return nil
}

func writeMetadataFile(filename string, dt any) error {
	b, err := json.MarshalIndent(dt, "", "  ")
	if err != nil {
		return err
	}
	return atomicwriter.WriteFile(filename, b, 0644)
}

// getImageID returns the image ID - the digest of the image config
func getImageID(resp map[string]string) string {
	dgst := resp[exptypes.ExporterImageDigestKey]
	if v, ok := resp[exptypes.ExporterImageConfigDigestKey]; ok {
		dgst = v
	}
	return dgst
}

func runBasicBuild(ctx context.Context, dockerCli command.Cli, opts *controllerapi.BuildOptions, printer *progress.Printer) (*client.SolveResponse, *build.Inputs, error) {
	resp, res, dfmap, err := cbuild.RunBuild(ctx, dockerCli, opts, dockerCli.In(), printer, false)
	if res != nil {
		res.Done()
	}
	return resp, dfmap, err
}

func runControllerBuild(ctx context.Context, dockerCli command.Cli, opts *controllerapi.BuildOptions, options buildOptions, printer *progress.Printer) (*client.SolveResponse, *build.Inputs, error) {
	if options.invokeConfig != nil && (options.dockerfileName == "-" || options.contextPath == "-") {
		// stdin must be usable for monitor
		return nil, nil, errors.Errorf("Dockerfile or context from stdin is not supported with invoke")
	}
	c, err := controller.NewController(ctx, options.ControlOptions, dockerCli, printer)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err := c.Close(); err != nil {
			logrus.Warnf("failed to close server connection %v", err)
		}
	}()

	// NOTE: buildx server has the current working directory different from the client
	// so we need to resolve paths to abosolute ones in the client.
	opts, err = controllerapi.ResolveOptionPaths(opts)
	if err != nil {
		return nil, nil, err
	}

	var ref string
	var retErr error
	var resp *client.SolveResponse
	var inputs *build.Inputs

	var f *ioset.SingleForwarder
	var pr io.ReadCloser
	var pw io.WriteCloser
	if options.invokeConfig == nil {
		pr = dockerCli.In()
	} else {
		f = ioset.NewSingleForwarder()
		f.SetReader(dockerCli.In())
		pr, pw = io.Pipe()
		f.SetWriter(pw, func() io.WriteCloser {
			pw.Close() // nolint:errcheck
			logrus.Debug("propagating stdin close")
			return nil
		})
	}

	ref, resp, inputs, err = c.Build(ctx, opts, pr, printer)
	if err != nil {
		var be *controllererrors.BuildError
		if errors.As(err, &be) {
			ref = be.SessionID
			retErr = err
			// We can proceed to monitor
		} else {
			return nil, nil, errors.Wrapf(err, "failed to build")
		}
	}

	if options.invokeConfig != nil {
		if err := pw.Close(); err != nil {
			logrus.Debug("failed to close stdin pipe writer")
		}
		if err := pr.Close(); err != nil {
			logrus.Debug("failed to close stdin pipe reader")
		}
	}

	if options.invokeConfig != nil && options.invokeConfig.needsDebug(retErr) {
		// Print errors before launching monitor
		if err := printError(retErr, printer); err != nil {
			logrus.Warnf("failed to print error information: %v", err)
		}

		pr2, pw2 := io.Pipe()
		f.SetWriter(pw2, func() io.WriteCloser {
			pw2.Close() // nolint:errcheck
			return nil
		})
		monitorBuildResult, err := options.invokeConfig.runDebug(ctx, ref, opts, c, pr2, os.Stdout, os.Stderr, printer)
		if err := pw2.Close(); err != nil {
			logrus.Debug("failed to close monitor stdin pipe reader")
		}
		if err != nil {
			logrus.Warnf("failed to run monitor: %v", err)
		}
		if monitorBuildResult != nil {
			// Update return values with the last build result from monitor
			resp, retErr = monitorBuildResult.Resp, monitorBuildResult.Err
		}
	} else {
		if err := c.Disconnect(ctx, ref); err != nil {
			logrus.Warnf("disconnect error: %v", err)
		}
	}

	return resp, inputs, retErr
}

func printError(err error, printer *progress.Printer) error {
	if err == nil {
		return nil
	}
	if err := printer.Pause(); err != nil {
		return err
	}
	defer printer.Unpause()
	for _, s := range errdefs.Sources(err) {
		s.Print(os.Stderr) // nolint:errcheck
	}
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err) // nolint:errcheck
	return nil
}

func listToMap(values []string, defaultEnv bool) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for _, value := range values {
		k, v, hasValue := strings.Cut(value, "=")
		if k == "" {
			return nil, errors.Errorf("invalid key-value pair %q: empty key", value)
		}
		if hasValue {
			result[k] = v
		} else if defaultEnv {
			if envVal, ok := os.LookupEnv(k); ok {
				result[k] = envVal
			}
		} else {
			result[k] = ""
		}
	}
	return result, nil
}

type invokeConfig struct {
	controllerapi.InvokeConfig
	onFlag     string
	invokeFlag string
}

func (cfg *invokeConfig) needsDebug(retErr error) bool {
	switch cfg.onFlag {
	case "always":
		return true
	case "error":
		return retErr != nil
	default:
		return cfg.invokeFlag != ""
	}
}

func (cfg *invokeConfig) runDebug(ctx context.Context, ref string, options *controllerapi.BuildOptions, c control.BuildxController, stdin io.ReadCloser, stdout io.WriteCloser, stderr console.File, progress *progress.Printer) (*monitor.MonitorBuildResult, error) {
	con := console.Current()
	if err := con.SetRaw(); err != nil {
		// TODO: run disconnect in build command (on error case)
		if err := c.Disconnect(ctx, ref); err != nil {
			logrus.Warnf("disconnect error: %v", err)
		}
		return nil, errors.Errorf("failed to configure terminal: %v", err)
	}
	defer con.Reset() // nolint:errcheck
	return monitor.RunMonitor(ctx, ref, options, &cfg.InvokeConfig, c, stdin, stdout, stderr, progress)
}

func dockerUlimitToControllerUlimit(u *dockeropts.UlimitOpt) *controllerapi.UlimitOpt {
	log.Printf("[DEBUG] ulimits: %#v", u)
	if u == nil {
		return &controllerapi.UlimitOpt{Values: map[string]*controllerapi.Ulimit{}}
	}
	values := make(map[string]*controllerapi.Ulimit)
	// TODO: commenting out the lines below is a workaround for the fact that the dockeropts.UlimitOpt returns a segmentation violation
	// when calling GetList() on a nil value. No idea how to fix this, yet

	// list := u.GetList()
	// if list == nil {
	// 	log.Printf("[WARN] GetList() returned nil")
	// 	return &controllerapi.UlimitOpt{Values: values}
	// }
	// for _, u := range u.GetList() {
	// 	values[u.Name] = &controllerapi.Ulimit{
	// 		Name: u.Name,
	// 		Hard: u.Hard,
	// 		Soft: u.Soft,
	// 	}
	// }
	return &controllerapi.UlimitOpt{Values: values}
}

func decodeExporterResponse(exporterResponse map[string]string) map[string]any {
	decFunc := func(k, v string) ([]byte, error) {
		if k == "result.json" {
			// result.json is part of metadata response for subrequests which
			// is already a JSON object: https://github.com/moby/buildkit/blob/f6eb72f2f5db07ddab89ac5e2bd3939a6444f4be/frontend/dockerui/requests.go#L100-L102
			return []byte(v), nil
		}
		return base64.StdEncoding.DecodeString(v)
	}
	out := make(map[string]any)
	for k, v := range exporterResponse {
		dt, err := decFunc(k, v)
		if err != nil {
			out[k] = v
			continue
		}
		var raw map[string]any
		if err = json.Unmarshal(dt, &raw); err != nil || len(raw) == 0 {
			var rawList []map[string]any
			if err = json.Unmarshal(dt, &rawList); err != nil || len(rawList) == 0 {
				out[k] = v
				continue
			}
		}
		out[k] = json.RawMessage(dt)
	}
	return out
}

func printWarnings(w io.Writer, warnings []client.VertexWarning, mode progressui.DisplayMode) {
	if len(warnings) == 0 || mode == progressui.QuietMode || mode == progressui.RawJSONMode {
		return
	}
	fmt.Fprintf(w, "\n ") // nolint:errcheck
	sb := &bytes.Buffer{}
	if len(warnings) == 1 {
		fmt.Fprintf(sb, "1 warning found") // nolint:errcheck
	} else {
		fmt.Fprintf(sb, "%d warnings found", len(warnings)) // nolint:errcheck
	}
	if logrus.GetLevel() < logrus.DebugLevel {
		fmt.Fprintf(sb, " (use docker --debug to expand)") // nolint:errcheck
	}
	fmt.Fprintf(sb, ":\n")                             // nolint:errcheck
	fmt.Fprint(w, aec.Apply(sb.String(), aec.YellowF)) // nolint:errcheck

	for _, warn := range warnings {
		fmt.Fprintf(w, " - %s\n", warn.Short) // nolint:errcheck
		if logrus.GetLevel() < logrus.DebugLevel {
			continue
		}
		for _, d := range warn.Detail {
			fmt.Fprintf(w, "%s\n", d) // nolint:errcheck
		}
		if warn.URL != "" {
			fmt.Fprintf(w, "More info: %s\n", warn.URL) // nolint:errcheck
		}
		if warn.SourceInfo != nil && warn.Range != nil {
			src := errdefs.Source{
				Info:   warn.SourceInfo,
				Ranges: warn.Range,
			}
			src.Print(w) // nolint:errcheck
		}
		fmt.Fprintf(w, "\n") // nolint:errcheck
	}
}

func printResult(w io.Writer, f *controllerapi.CallFunc, res map[string]string, target string, inp *build.Inputs) (int, error) {
	switch f.Name {
	case "outline":
		return 0, printValue(w, outline.PrintOutline, outline.SubrequestsOutlineDefinition.Version, f.Format, res)
	case "targets":
		return 0, printValue(w, targets.PrintTargets, targets.SubrequestsTargetsDefinition.Version, f.Format, res)
	case "subrequests.describe":
		return 0, printValue(w, subrequests.PrintDescribe, subrequests.SubrequestsDescribeDefinition.Version, f.Format, res)
	case "lint":
		lintResults := lint.LintResults{}
		if result, ok := res["result.json"]; ok {
			if err := json.Unmarshal([]byte(result), &lintResults); err != nil {
				return 0, err
			}
		}

		warningCount := len(lintResults.Warnings)
		if f.Format != "json" && warningCount > 0 {
			var warningCountMsg string
			if warningCount == 1 {
				warningCountMsg = "1 warning has been found!"
			} else if warningCount > 1 {
				warningCountMsg = fmt.Sprintf("%d warnings have been found!", warningCount)
			}
			fmt.Fprintf(w, "Check complete, %s\n", warningCountMsg) // nolint:errcheck
		}
		sourceInfoMap := func(sourceInfo *solverpb.SourceInfo) *solverpb.SourceInfo {
			if sourceInfo == nil || inp == nil {
				return sourceInfo
			}
			if target == "" {
				target = "default"
			}

			if inp.DockerfileMappingSrc != "" {
				newSourceInfo := proto.Clone(sourceInfo).(*solverpb.SourceInfo)
				newSourceInfo.Filename = inp.DockerfileMappingSrc
				return newSourceInfo
			}
			return sourceInfo
		}

		printLintWarnings := func(dt []byte, w io.Writer) error {
			return lintResults.PrintTo(w, sourceInfoMap)
		}

		err := printValue(w, printLintWarnings, lint.SubrequestLintDefinition.Version, f.Format, res)
		if err != nil {
			return 0, err
		}

		if lintResults.Error != nil {
			// Print the error message and the source
			// Normally, we would use `errdefs.WithSource` to attach the source to the
			// error and let the error be printed by the handling that's already in place,
			// but here we want to print the error in a way that's consistent with how
			// the lint warnings are printed via the `lint.PrintLintViolations` function,
			// which differs from the default error printing.
			if f.Format != "json" && len(lintResults.Warnings) > 0 {
				fmt.Fprintln(w) // nolint:errcheck
			}
			lintBuf := bytes.NewBuffer(nil)
			lintResults.PrintErrorTo(lintBuf, sourceInfoMap)
			return 0, errors.New(lintBuf.String())
		} else if len(lintResults.Warnings) == 0 && f.Format != "json" {
			fmt.Fprintln(w, "Check complete, no warnings found.") // nolint:errcheck
		}
	default:
		if dt, ok := res["result.json"]; ok && f.Format == "json" {
			fmt.Fprintln(w, dt) // nolint:errcheck
		} else if dt, ok := res["result.txt"]; ok {
			fmt.Fprint(w, dt) // nolint:errcheck
		} else {
			fmt.Fprintf(w, "%s %+v\n", f, res) // nolint:errcheck
		}
	}
	if v, ok := res["result.statuscode"]; !f.IgnoreStatus && ok {
		if n, err := strconv.Atoi(v); err == nil && n != 0 {
			return n, nil
		}
	}
	return 0, nil
}

type callFunc func([]byte, io.Writer) error

func printValue(w io.Writer, printer callFunc, version string, format string, res map[string]string) error {
	if format == "json" {
		fmt.Fprintln(w, res["result.json"]) // nolint:errcheck
		return nil
	}

	if res["version"] != "" && versions.LessThan(version, res["version"]) && res["result.txt"] != "" {
		// structure is too new and we don't know how to print it
		fmt.Fprint(w, res["result.txt"]) // nolint:errcheck
		return nil
	}
	return printer([]byte(res["result.json"]), w)
}
