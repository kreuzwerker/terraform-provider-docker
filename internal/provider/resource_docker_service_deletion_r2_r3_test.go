//go:build linux

package provider

// Native R2/R3 only. No init/TestMain, FromEnv, Configure, or subprocesses.
// The external, separately guarded launcher owns all newly created VMs.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type nativeB23Node struct {
	Endpoint string
	PID      int
	Address  string
}
type nativeB23Manifest struct {
	Engine  string
	Nonce   string
	Slice   string
	Manager nativeB23Node
	Worker  nativeB23Node
}

// Filesystem, environment, endpoint and live process guards BEFORE any client.
func nativeB23Guard(t *testing.T, allowedSlice string) (string, nativeB23Manifest) {
	t.Helper()
	if os.Getenv("DOCKER_SERVICE_B23") != "new-isolated-vms-only" {
		t.Skip("B23 requires the explicit new isolated-VM launcher")
	}
	s := os.Getenv("DOCKER_SERVICE_B23_SANDBOX")
	if !strings.HasPrefix(s, "/tmp/docker-service-b-r23.") || filepath.Clean(s) != s {
		t.Fatal("invalid B23 sandbox")
	}
	real, err := filepath.EvalSymlinks(s)
	st, statErr := os.Stat(s)
	if err != nil || real != s || statErr != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		t.Fatal("sandbox must be real and private")
	}
	if stat, ok := st.Sys().(*syscall.Stat_t); !ok || stat.Uid != uint32(os.Getuid()) {
		t.Fatal("sandbox not owned by current user")
	}
	for key, suffix := range map[string]string{"HOME": "home", "TMPDIR": "tmp", "GOPATH": "go", "GOCACHE": "go-cache", "DOCKER_CONFIG": "docker-config", "DOCKER_HOST": "does-not-exist.sock"} {
		want := filepath.Join(s, suffix)
		if key == "DOCKER_HOST" {
			want = "unix://" + want
		}
		if os.Getenv(key) != want {
			t.Fatalf("unclean %s", key)
		}
	}
	if _, err := os.Lstat(filepath.Join(s, "does-not-exist.sock")); !os.IsNotExist(err) {
		t.Fatal("Docker host socket must not exist")
	}
	if os.Getenv("GOWORK") != "off" || os.Getenv("GOTOOLCHAIN") != "local" || os.Getenv("CGO_ENABLED") != "0" || os.Getenv("TF_ACC") != "" || os.Getenv("DOCKER_CONTEXT") != "" || os.Getenv("DOCKER_API_VERSION") != "" || os.Getenv("SSH_AUTH_SOCK") != "" {
		t.Fatal("unclean execution environment")
	}
	guard, err := os.ReadFile(filepath.Join(s, "guard"))
	if err != nil || string(guard) != "docker-service-deletion-b23-isolated-v1\n" {
		t.Fatal("missing isolation marker")
	}
	var m nativeB23Manifest
	data, err := os.ReadFile(filepath.Join(s, "manifest.json"))
	if err != nil || json.Unmarshal(data, &m) != nil || m.Engine != "29.9.1" || len(m.Nonce) != 24 || m.Slice != allowedSlice || os.Getenv("DOCKER_SERVICE_B23_SLICE") != m.Slice {
		t.Fatal("invalid fresh-VM manifest/slice")
	}
	seen := map[string]bool{}
	for i, n := range []nativeB23Node{m.Manager, m.Worker} {
		u, err := url.Parse(n.Endpoint)
		if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Path != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			t.Fatal("non-isolated endpoint")
		}
		port, err := net.LookupPort("tcp", u.Port())
		if err != nil || port < 1024 || seen[u.Host] || n.PID <= 1 || n.Address != fmt.Sprintf("192.0.2.%d", i+1) {
			t.Fatal("invalid node/endpoint")
		}
		seen[u.Host] = true
		cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", n.PID))
		owned, ownerErr := os.Stat(fmt.Sprintf("/proc/%d", n.PID))
		if err != nil || ownerErr != nil || !strings.Contains(string(cmd), "qemu-system-x86_64") || !bytes.Contains(cmd, []byte(s+"/vm-"+m.Slice+"-")) || !bytes.Contains(cmd, []byte("hostfwd=tcp:127.0.0.1:"+u.Port()+"-:2375")) || !bytes.Contains(cmd, []byte("restrict=on")) {
			t.Fatal("endpoint lacks its live new isolated QEMU process")
		}
		if stat, ok := owned.Sys().(*syscall.Stat_t); !ok || stat.Uid != uint32(os.Getuid()) {
			t.Fatal("QEMU not user-owned")
		}
	}
	return s, m
}

type nativeB23Event struct {
	At     time.Time
	End    time.Time `json:",omitempty"`
	Kind   string
	Method string `json:",omitempty"`
	Path   string `json:",omitempty"`
	Status int    `json:",omitempty"`
	Body   string `json:",omitempty"`
	TaskID string `json:",omitempty"`
	Error  string `json:",omitempty"`
}
type nativeB23Transport struct {
	host   string
	base   *http.Transport
	before func(*http.Request) error
	tap    func(*http.Request, *http.Response, []byte, time.Time) error
}

func (r *nativeB23Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "http" || req.URL.Host != r.host {
		return nil, fmt.Errorf("B23 blocked endpoint")
	}
	if r.before != nil {
		if err := r.before(req); err != nil {
			return nil, err
		}
	}
	resp, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if r.tap != nil {
		if err := r.tap(req, resp, body, time.Now()); err != nil {
			return nil, err
		}
	}
	resp.Body = io.NopCloser(bytes.NewReader(body)) // identical native bytes
	return resp, nil
}
func nativeB23Client(t *testing.T, endpoint string, before func(*http.Request) error, tap func(*http.Request, *http.Response, []byte, time.Time) error) *client.Client {
	t.Helper()
	u, _ := url.Parse(endpoint) // guarded exact endpoint
	base := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{}).DialContext}
	rt := &nativeB23Transport{host: u.Host, base: base, before: before, tap: tap}
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+u.Host), client.WithHTTPClient(&http.Client{Transport: rt, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("B23 redirects forbidden") }}), client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close(); base.CloseIdleConnections() })
	return cli
}
func nativeB23Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

// A real, cancelable test-only observation barrier. Its cap never changes the
// helper's clock/context or supplies a task observation to production.
func nativeB23Pause(ctx context.Context, cap time.Duration, sample func(context.Context) (bool, error)) (bool, error) {
	pauseCtx, cancel := context.WithTimeout(ctx, cap)
	defer cancel()
	finish := func(err error) (bool, error) {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if errors.Is(pauseCtx.Err(), context.DeadlineExceeded) {
			return false, nil
		}
		return false, err
	}
	for {
		if err := pauseCtx.Err(); err != nil {
			return finish(err)
		}
		done, err := sample(pauseCtx)
		if err != nil {
			return finish(err)
		}
		if done {
			return true, nil
		}
		if err := nativeB23Wait(pauseCtx, 50*time.Millisecond); err != nil {
			return finish(err)
		}
	}
}

func nativeB23IDs(tasks []swarm.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	sort.Strings(ids)
	return ids
}
func nativeB23Path(req *http.Request) string {
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	if len(parts) == 2 && strings.HasPrefix(parts[0], "v1.") {
		return "/" + parts[1]
	}
	return req.URL.Path
}
func nativeB23Cluster(t *testing.T, ctx context.Context, m nativeB23Manifest) (*client.Client, *client.Client, string) {
	t.Helper()
	manager := nativeB23Client(t, m.Manager.Endpoint, nil, nil)
	worker := nativeB23Client(t, m.Worker.Endpoint, nil, nil) // fixture/oracle only
	for name, cli := range map[string]*client.Client{"manager": manager, "worker": worker} {
		info, err := cli.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if info.ServerVersion != m.Engine || !strings.Contains(strings.Join(info.Labels, "\n"), "b23.nonce="+m.Nonce) || info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive {
			t.Fatal("not the fresh labeled engine")
		}
		v, err := cli.ServerVersion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s engine=%s serverAPI=%s minAPI=%s negotiatedAPI=%s os=%s kernel=%s commit=%s", name, v.Version, v.APIVersion, v.MinAPIVersion, cli.ClientVersion(), v.Os, v.KernelVersion, v.GitCommit)
	}
	_, err := manager.SwarmInit(ctx, swarm.InitRequest{ListenAddr: m.Manager.Address + ":2377", AdvertiseAddr: m.Manager.Address})
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := manager.SwarmInspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.SwarmJoin(ctx, swarm.JoinRequest{ListenAddr: m.Worker.Address + ":2377", AdvertiseAddr: m.Worker.Address, RemoteAddrs: []string{m.Manager.Address + ":2377"}, JoinToken: cluster.JoinTokens.Worker}); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		info, err := worker.Info(ctx)
		if err == nil && info.Swarm.NodeID != "" {
			node, _, err := manager.NodeInspectWithRaw(ctx, info.Swarm.NodeID)
			if err == nil && node.Status.State == swarm.NodeStateReady {
				return manager, worker, info.Swarm.NodeID
			}
		}
		if err := nativeB23Wait(ctx, 100*time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("fresh worker not ready")
	return nil, nil, ""
}

func TestDockerServiceDeletionNativeR2(t *testing.T) {
	slice := os.Getenv("DOCKER_SERVICE_B23_SLICE")
	if slice != "global" && slice != "pending" {
		slice = "global"
	} // guard must still run/skip before any client
	s, m := nativeB23Guard(t, slice)
	nativeB23Run(t, s, m)
}

func TestDockerServiceDeletionNativeR3(t *testing.T) {
	s, m := nativeB23Guard(t, "metadata")
	// Exercise the very barrier used below, with a real cancellation, before
	// clients/VM endpoints are touched by this test.
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(20*time.Millisecond, cancel)
	started := time.Now()
	done, err := nativeB23Pause(ctx, 25*time.Second, func(context.Context) (bool, error) { return false, nil })
	timer.Stop()
	cancel()
	if done || !errors.Is(err, context.Canceled) || time.Since(started) > time.Second {
		t.Fatalf("observation pause not cancelable: %v", err)
	}
	t.Logf("real pause cancellation verified in %s", time.Since(started))
	nativeB23Run(t, s, m)
}

func nativeB23Run(t *testing.T, s string, m nativeB23Manifest) {
	ctx, cancel := context.WithTimeout(context.Background(), 210*time.Second)
	defer cancel()
	manager, worker, workerID := nativeB23Cluster(t, ctx, m)
	grace := 10 * time.Second
	spec := swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "b23-" + m.Nonce + "-" + m.Slice}, Mode: swarm.ServiceMode{Global: &swarm.GlobalService{}}, TaskTemplate: swarm.TaskSpec{
		ContainerSpec: &swarm.ContainerSpec{Image: "b23-synthetic:busybox", Command: []string{"/bin/sh", "-c", `trap 'echo B23_TERM; sleep 4; echo B23_EXIT; exit 0' TERM; echo B23_READY; while :; do sleep 0.1; done`}, StopGracePeriod: &grace},
		Placement:     &swarm.Placement{Constraints: []string{"node.id==" + workerID}}, RestartPolicy: &swarm.RestartPolicy{Condition: swarm.RestartPolicyConditionNone},
	}}
	if m.Slice == "pending" {
		replicas := uint64(1)
		spec.Mode = swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}
		// Immutable from creation: neither fresh node has this synthetic label.
		spec.TaskTemplate.Placement.Constraints = []string{"node.labels.b23-never==" + m.Nonce}
	}
	created, err := manager.ServiceCreate(ctx, spec, swarm.ServiceCreateOptions{QueryRegistry: false})
	if err != nil {
		t.Fatal(err)
	}
	sid := created.ID
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = manager.ServiceRemove(c, sid)
	}()
	var before []swarm.Task
	ready := false
	for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
		before, err = manager.TaskList(ctx, swarm.TaskListOptions{Filters: filters.NewArgs(filters.Arg("service", sid))})
		ready = err == nil && len(before) == 1
		for _, task := range before {
			if m.Slice == "pending" {
				ready = ready && task.ID != "" && task.ServiceID == sid && task.NodeID == "" && task.Status.State == swarm.TaskStatePending && deletionContainerID(task) == ""
			} else {
				ready = ready && task.ID != "" && task.ServiceID == sid && task.NodeID == workerID && task.Status.State == swarm.TaskStateRunning && deletionContainerID(task) != ""
			}
		}
		if ready {
			break
		}
		if err := nativeB23Wait(ctx, 100*time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	if !ready {
		b, _ := json.Marshal(before)
		t.Fatalf("premise not exercised, fixture not ready: %s (%v)", b, err)
	}
	tracePath := filepath.Join(s, "results", m.Slice+".jsonl")
	trace, err := os.OpenFile(tracePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := trace.Close(); err != nil {
			t.Errorf("closing native trace: %v", err)
		}
	}()
	enc := json.NewEncoder(trace)
	emit := func(e nativeB23Event) {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	known := map[string]swarm.Task{}
	for _, task := range before {
		known[task.ID] = task
		if m.Slice == "pending" {
			continue
		} // no empty-CID oracle or inference
		c, err := worker.ContainerInspect(ctx, deletionContainerID(task))
		if err != nil || c.State == nil || !c.State.Running {
			t.Fatalf("independent running oracle: %v", err)
		}
		b, _ := json.Marshal(c.State)
		emit(nativeB23Event{At: time.Now(), Kind: "oracle-before", TaskID: task.ID, Body: string(b)})
	}
	captured, deleted := false, false
	proofs := map[string]time.Time{}
	postReads, nativeMissing := map[string]int{}, map[string]int{}
	var firstDelete, resumed time.Time
	paused, auxGC := false, map[string]bool{}
	oracleExit := map[string]bool{}
	// This client belongs only to the test. Its observations are recorded in
	// a separate channel, never returned as a helper TaskInspect response.
	var auxiliary *client.Client
	if m.Slice == "metadata" {
		auxiliary = nativeB23Client(t, m.Manager.Endpoint, nil, func(req *http.Request, resp *http.Response, body []byte, at time.Time) error {
			emit(nativeB23Event{At: at, Kind: "auxiliary-http", Method: req.Method, Path: req.URL.RequestURI(), Status: resp.StatusCode, Body: string(body)})
			return nil
		})
	}
	pauseSample := func(c context.Context) (bool, error) {
		for id, task := range known {
			if !oracleExit[id] {
				start := time.Now()
				containerInfo, err := worker.ContainerInspect(c, deletionContainerID(task))
				if err != nil {
					emit(nativeB23Event{At: start, End: time.Now(), Kind: "oracle-paused-sample", TaskID: id, Error: err.Error()})
					return false, fmt.Errorf("paused independent exit oracle: %w", err)
				}
				b, _ := json.Marshal(containerInfo.State)
				emit(nativeB23Event{At: start, End: time.Now(), Kind: "oracle-paused-sample", TaskID: id, Body: string(b)})
				state := containerInfo.State
				if state != nil && !state.Running {
					if state.ExitCode != 0 || state.FinishedAt == "" {
						return false, fmt.Errorf("independent exit not successful")
					}
					logs, err := worker.ContainerLogs(c, containerInfo.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
					if err != nil {
						return false, err
					}
					var stdout, stderr bytes.Buffer
					_, err = stdcopy.StdCopy(&stdout, &stderr, logs)
					_ = logs.Close()
					text := stdout.String() + stderr.String()
					emit(nativeB23Event{At: time.Now(), Kind: "oracle-paused-exit-logs", TaskID: id, Body: text})
					if err != nil || !strings.Contains(text, "B23_READY\nB23_TERM\nB23_EXIT\n") {
						return false, fmt.Errorf("independent delayed exit logs missing: %q (%v)", text, err)
					}
					oracleExit[id] = true
				}
			}
			if !auxGC[id] {
				_, _, err := auxiliary.TaskInspectWithRaw(c, id)
				if serviceDeletionNotFound(err, "task", id) {
					auxGC[id] = true
				} else if err != nil {
					return false, err
				}
			}
		}
		return len(oracleExit) == len(known) && len(auxGC) > 0, nil
	}
	beforeRequest := func(req *http.Request) error {
		path := nativeB23Path(req)
		allowed := path == "/_ping" && (req.Method == "HEAD" || req.Method == "GET") || path == "/services/"+sid && req.Method == "GET"
		if path == "/tasks" && req.Method == "GET" && !deleted {
			var f map[string]map[string]bool
			allowed = !captured && json.Unmarshal([]byte(req.URL.Query().Get("filters")), &f) == nil && len(f) == 1 && len(f["service"]) == 1 && f["service"][sid]
		}
		if path == "/services/"+sid && req.Method == "DELETE" && captured && !deleted {
			allowed, deleted, firstDelete = true, true, time.Now()
		}
		if strings.HasPrefix(path, "/tasks/") && req.Method == "GET" {
			_, allowed = known[strings.TrimPrefix(path, "/tasks/")]
		}
		if !allowed {
			return fmt.Errorf("B23 blocked helper route before network: %s %s", req.Method, req.URL.RequestURI())
		}
		if m.Slice == "metadata" && deleted && strings.HasPrefix(path, "/tasks/") && !paused {
			paused = true
			if len(proofs) != 0 {
				return fmt.Errorf("R3 already received positive task proof before pause")
			}
			deadline, _ := req.Context().Deadline()
			emit(nativeB23Event{At: time.Now(), Kind: "pause-begin", Body: fmt.Sprintf("firstDelete=%s phaseDeadline=%s cap=25s", firstDelete.Format(time.RFC3339Nano), deadline.Format(time.RFC3339Nano))})
			gc, err := nativeB23Pause(req.Context(), 25*time.Second, pauseSample)
			resumed = time.Now()
			emit(nativeB23Event{At: resumed, Kind: "pause-end", Body: fmt.Sprintf("gc=%t oracleExit=%v auxGC=%v", gc, oracleExit, auxGC)})
			if err != nil {
				return err
			}
		}
		emit(nativeB23Event{At: time.Now(), Kind: "helper-request", Method: req.Method, Path: req.URL.RequestURI()})
		return nil
	}
	// Separate native task assertions; no helper record or oracle result is
	// supplied to production. Empty ContainerID and 404 never supply proof.
	nativeTaskProof := func(req *http.Request, body []byte, received time.Time) error {
		id := strings.TrimPrefix(nativeB23Path(req), "/tasks/")
		initial := known[id]
		var task swarm.Task
		if err := json.Unmarshal(body, &task); err != nil {
			return err
		}
		if m.Slice == "pending" {
			if task.ID != id || task.ServiceID != sid || task.Version.Index < initial.Version.Index || task.NodeID != "" || deletionContainerID(task) != "" {
				return fmt.Errorf("pending constraint escaped or identity changed; no exit oracle satisfied")
			}
			if task.Status.State == swarm.TaskStateShutdown || task.Status.State == swarm.TaskStateComplete || task.Status.State == swarm.TaskStateRejected {
				proofs[id] = received
			}
			return nil
		}
		if task.Status.State != swarm.TaskStateShutdown && task.Status.State != swarm.TaskStateComplete {
			return nil
		}
		if !deleted || task.ID != id || task.ServiceID != sid || task.NodeID != initial.NodeID || deletionContainerID(task) != deletionContainerID(initial) || task.Version.Index < initial.Version.Index {
			return fmt.Errorf("incoherent native proof")
		}
		start := time.Now()
		c, err := worker.ContainerInspect(req.Context(), deletionContainerID(initial))
		if err != nil {
			return fmt.Errorf("independent exit oracle: %w", err)
		}
		b, _ := json.Marshal(c.State)
		emit(nativeB23Event{At: start, End: time.Now(), Kind: "oracle-proof", TaskID: id, Body: string(b)})
		if c.State == nil || c.State.Running || c.State.ExitCode != 0 || c.State.FinishedAt == "" {
			return fmt.Errorf("native proof precedes independent exit")
		}
		logs, err := worker.ContainerLogs(req.Context(), c.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			return err
		}
		var stdout, stderr bytes.Buffer
		_, copyErr := stdcopy.StdCopy(&stdout, &stderr, logs)
		_ = logs.Close()
		text := stdout.String() + stderr.String()
		emit(nativeB23Event{At: time.Now(), Kind: "oracle-logs", TaskID: id, Body: text})
		if copyErr != nil || !strings.Contains(text, "B23_READY\nB23_TERM\nB23_EXIT\n") {
			return fmt.Errorf("delayed exit logs missing: %q (%v)", text, copyErr)
		}
		proofs[id] = received
		return nil
	}
	tap := func(req *http.Request, resp *http.Response, body []byte, received time.Time) error {
		defer func() {
			emit(nativeB23Event{At: received, End: time.Now(), Kind: "helper-tap-delay", Method: req.Method, Path: req.URL.RequestURI()})
		}()
		path := nativeB23Path(req)
		emit(nativeB23Event{At: received, Kind: "helper-http", Method: req.Method, Path: req.URL.RequestURI(), Status: resp.StatusCode, Body: string(body)})
		switch {
		case path == "/tasks" && req.Method == "GET" && !deleted:
			var tasks []swarm.Task
			if resp.StatusCode != 200 || json.Unmarshal(body, &tasks) != nil || !reflect.DeepEqual(nativeB23IDs(tasks), nativeB23IDs(before)) {
				return fmt.Errorf("capture differs from complete fixture")
			}
			for _, task := range tasks {
				if m.Slice == "pending" {
					if task.ID == "" || task.ServiceID != sid || task.Status.State != swarm.TaskStatePending || task.NodeID != "" || deletionContainerID(task) != "" {
						return fmt.Errorf("helper did not actually capture pending TaskID without ContainerID")
					}
				} else if task.Status.State != swarm.TaskStateRunning || task.NodeID != workerID || deletionContainerID(task) == "" {
					return fmt.Errorf("helper capture not RUNNING remotely")
				}
			}
			captured = true
		case strings.HasPrefix(path, "/tasks/") && req.Method == "GET":
			id := strings.TrimPrefix(path, "/tasks/")
			if deleted {
				postReads[id]++
			}
			if deleted && resp.StatusCode == 404 {
				var e struct {
					Message string `json:"message"`
				}
				if json.Unmarshal(body, &e) == nil && e.Message == "task "+id+" not found" {
					nativeMissing[id]++
				}
			}
			if resp.StatusCode != 200 {
				return nil
			}
			return nativeTaskProof(req, body, received)
		case path == "/services/"+sid && req.Method == "GET" && resp.StatusCode == 200:
			var service swarm.Service
			if json.Unmarshal(body, &service) != nil || service.ID != sid {
				return fmt.Errorf("service identity mismatch")
			}
			if m.Slice == "pending" {
				if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Global != nil || service.Spec.TaskTemplate.Placement == nil || !reflect.DeepEqual(service.Spec.TaskTemplate.Placement.Constraints, spec.TaskTemplate.Placement.Constraints) {
					return fmt.Errorf("pending fixture mutated")
				}
			} else if service.Spec.Mode.Global == nil || service.Spec.Mode.Replicated != nil {
				return fmt.Errorf("not global from creation")
			}
		}
		return nil
	}
	helper := nativeB23Client(t, m.Manager.Endpoint, beforeRequest, tap)
	helperCtx, stop := context.WithTimeout(ctx, 60*time.Second)
	defer stop()
	started := time.Now()
	err = deleteService(helperCtx, sid, helper)
	returned := time.Now()
	message := ""
	if err != nil {
		message = err.Error()
	}
	emit(nativeB23Event{At: returned, Kind: "helper-return", Error: message})
	t.Logf("engine=%s slice=%s S=%s T=%v negotiatedAPI=%s elapsed=%s delete-to-return=%s proofs=%d trace=%s error=%v", m.Engine, m.Slice, sid, nativeB23IDs(before), helper.ClientVersion(), returned.Sub(started), returned.Sub(firstDelete), len(proofs), tracePath, err)
	if m.Slice == "metadata" {
		if !paused || len(oracleExit) != len(known) || len(auxGC) == 0 {
			t.Fatalf("R3 NOT EXERCISED: no native GC with independent exit within bounded pause; helper=%v", err)
		}
		if resumed.IsZero() || resumed.Sub(firstDelete) > 30*time.Second {
			t.Fatal("pause consumed observation window instead of exercising GC")
		}
		if err == nil || !errors.Is(err, context.DeadlineExceeded) || helperCtx.Err() != nil || len(proofs) != 0 || !strings.Contains(message, "object=removed, capture_complete=true, captured=1, satisfied=0") {
			t.Fatalf("R3 not expected lost-proof unknown: %v", err)
		}
		for id := range auxGC {
			if nativeMissing[id] == 0 || postReads[id] == 0 || !strings.Contains(message, id) {
				t.Fatal("helper did not itself observe native metadata absence after resume")
			}
		}
		t.Logf("R3 safety PASS: independent exit, native missing=%v postreads=%v pause=%s; unknown is NOT B success", nativeMissing, postReads, resumed.Sub(firstDelete))
	} else if m.Slice == "pending" && err != nil {
		if !captured || !deleted || len(proofs) != 0 || !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(message, "object=removed, capture_complete=true, captured=1, satisfied=0") {
			t.Fatalf("not the expected unresolved-task unknown: %v", err)
		}
		for id := range known {
			if postReads[id] == 0 || nativeMissing[id] == 0 || !strings.Contains(message, id) {
				t.Fatalf("pending TaskID omitted instead of retained unknown: %s", id)
			}
		}
		t.Logf("R2 pending safety PASS: native missing=%v postreads=%v; unknown is NOT B success or proof of never executing", nativeMissing, postReads)
	} else {
		if err != nil {
			t.Fatal(err)
		}
		if !deleted || !captured || len(proofs) != len(before) {
			t.Fatal("success without native/independent proof for all T")
		}
	}
	for _, at := range proofs {
		if at.After(returned) {
			t.Fatal("success before task proof")
		}
	}
	if m.Slice != "pending" && returned.Sub(firstDelete) < 4*time.Second {
		t.Fatal("delayed exit not exercised")
	}
	if _, _, err := manager.ServiceInspectWithRaw(ctx, sid, swarm.ServiceInspectOptions{}); !serviceDeletionNotFound(err, "service", sid) {
		t.Fatalf("independent S absence: %v", err)
	}
}
