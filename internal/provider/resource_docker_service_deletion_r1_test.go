//go:build linux

package provider

// R1 only: opt-in native fixture. No init, TestMain, environment Docker client,
// provider configuration, or subprocesses. The external launcher owns new VMs.
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
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

type nativeR1Node struct {
	Endpoint string
	PID      int
	Address  string
}

type nativeR1Manifest struct {
	Engine  string
	Nonce   string
	Manager nativeR1Node
	Worker  nativeR1Node
}

// All filesystem/endpoint/process checks precede construction of ANY client.
func nativeR1Guard(t *testing.T) (string, nativeR1Manifest) {
	t.Helper()
	if os.Getenv("DOCKER_SERVICE_R1") != "new-isolated-vms-only" {
		t.Skip("R1 requires the explicit isolated-VM launcher")
	}
	s := os.Getenv("DOCKER_SERVICE_R1_SANDBOX")
	if !strings.HasPrefix(s, "/tmp/docker-service-b-r1.") || filepath.Clean(s) != s {
		t.Fatal("invalid R1 sandbox")
	}
	real, err := filepath.EvalSymlinks(s)
	if err != nil || real != s {
		t.Fatal("sandbox must be real, not symlinked")
	}
	st, err := os.Stat(s)
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		t.Fatal("sandbox must be private")
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
		t.Fatal("DOCKER_HOST is not nonexistent")
	}
	if os.Getenv("GOWORK") != "off" || os.Getenv("GOTOOLCHAIN") != "local" || os.Getenv("TF_ACC") != "" || os.Getenv("DOCKER_CONTEXT") != "" || os.Getenv("DOCKER_API_VERSION") != "" {
		t.Fatal("unexpected execution environment")
	}
	guard, err := os.ReadFile(filepath.Join(s, "guard"))
	if err != nil || string(guard) != "docker-service-deletion-r1-isolated-v1\n" {
		t.Fatal("missing isolation marker")
	}
	var m nativeR1Manifest
	data, err := os.ReadFile(filepath.Join(s, "manifest.json"))
	if err != nil || json.Unmarshal(data, &m) != nil || len(m.Nonce) != 24 || (m.Engine != "28.0.4" && m.Engine != "29.9.1") {
		t.Fatal("invalid new-VM manifest")
	}
	seen := map[string]bool{}
	for i, n := range []nativeR1Node{m.Manager, m.Worker} {
		u, err := url.Parse(n.Endpoint)
		if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Path != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			t.Fatal("non-isolated endpoint")
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1024 || port > 65535 || seen[u.Host] || n.PID <= 1 || n.Address != fmt.Sprintf("192.0.2.%d", i+1) {
			t.Fatal("invalid endpoint or node")
		}
		seen[u.Host] = true
		cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", n.PID))
		owned, ownerErr := os.Stat(fmt.Sprintf("/proc/%d", n.PID))
		if err != nil || ownerErr != nil || !strings.Contains(string(cmd), "qemu-system-x86_64") || !bytes.Contains(cmd, []byte(s+"/vm-"+m.Engine)) {
			t.Fatal("endpoint lacks its live new QEMU process")
		}
		if stat, ok := owned.Sys().(*syscall.Stat_t); !ok || stat.Uid != uint32(os.Getuid()) {
			t.Fatal("QEMU is not owned by this user")
		}
	}
	return s, m
}

type nativeR1Event struct {
	At     time.Time
	End    time.Time
	Kind   string
	Method string `json:",omitempty"`
	Path   string `json:",omitempty"`
	Status int    `json:",omitempty"`
	Body   string `json:",omitempty"`
	TaskID string `json:",omitempty"`
	Error  string `json:",omitempty"`
}

type nativeR1Transport struct {
	host   string
	base   *http.Transport
	before func(*http.Request) error
	tap    func(*http.Request, *http.Response, []byte, time.Time) error
}

func (r *nativeR1Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "http" || req.URL.Host != r.host {
		return nil, fmt.Errorf("R1 blocked endpoint %s", req.URL.Host)
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
	received := time.Now()
	if r.tap != nil {
		if err := r.tap(req, resp, body, received); err != nil {
			return nil, err
		}
	}
	resp.Body = io.NopCloser(bytes.NewReader(body)) // unchanged native bytes
	return resp, nil
}

func nativeR1Client(t *testing.T, endpoint string, before func(*http.Request) error, tap func(*http.Request, *http.Response, []byte, time.Time) error) *client.Client {
	t.Helper()
	u, _ := url.Parse(endpoint) // guard already checked the exact endpoint
	base := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{}).DialContext}
	rt := &nativeR1Transport{host: u.Host, base: base, before: before, tap: tap}
	cli, err := client.NewClientWithOpts(client.WithHost("tcp://"+u.Host), client.WithHTTPClient(&http.Client{Transport: rt, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("R1 forbids redirects") }}), client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close(); base.CloseIdleConnections() })
	return cli
}

func nativeR1IDs(tasks []swarm.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	sort.Strings(ids)
	return ids
}

func TestDockerServiceDeletionNativeR1(t *testing.T) {
	s, m := nativeR1Guard(t)
	manager := nativeR1Client(t, m.Manager.Endpoint, nil, nil)
	worker := nativeR1Client(t, m.Worker.Endpoint, nil, nil) // fixture setup/oracle ONLY
	ctx, cancel := context.WithTimeout(context.Background(), 210*time.Second)
	defer cancel()
	for name, cli := range map[string]*client.Client{"manager": manager, "worker": worker} {
		info, err := cli.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if info.ServerVersion != m.Engine || !strings.Contains(strings.Join(info.Labels, "\n"), "r1.nonce="+m.Nonce) || info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive {
			t.Fatalf("%s is not the fresh labeled engine", name)
		}
		v, err := cli.ServerVersion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s engine=%s serverAPI=%s minAPI=%s negotiatedAPI=%s os=%s kernel=%s commit=%s", name, v.Version, v.APIVersion, v.MinAPIVersion, cli.ClientVersion(), v.Os, v.KernelVersion, v.GitCommit)
	}
	managerID, err := manager.SwarmInit(ctx, swarm.InitRequest{ListenAddr: m.Manager.Address + ":2377", AdvertiseAddr: m.Manager.Address})
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
	var workerID string
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		info, err := worker.Info(ctx)
		if err == nil && info.Swarm.NodeID != "" {
			workerID = info.Swarm.NodeID
			node, _, inspectErr := manager.NodeInspectWithRaw(ctx, workerID)
			if inspectErr == nil && node.Status.State == swarm.NodeStateReady {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if workerID == "" {
		t.Fatal("new worker did not join")
	}
	for _, location := range []struct {
		name string
		id   string
		cli  *client.Client
	}{{"local", managerID, manager}, {"remote", workerID, worker}} {
		if !t.Run(location.name, func(t *testing.T) {
			grace, replicas := 10*time.Second, uint64(2)
			spec := swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "r1-" + m.Nonce + "-" + location.name}, Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}, TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{Image: "r1-synthetic:busybox", Command: []string{"/bin/sh", "-c", `trap 'echo R1_TERM; sleep 4; echo R1_EXIT; exit 0' TERM; echo R1_READY; while :; do sleep 0.1; done`}, StopGracePeriod: &grace},
				Placement:     &swarm.Placement{Constraints: []string{"node.id==" + location.id}}, RestartPolicy: &swarm.RestartPolicy{Condition: swarm.RestartPolicyConditionNone},
			}}
			created, err := manager.ServiceCreate(ctx, spec, swarm.ServiceCreateOptions{QueryRegistry: false})
			if err != nil {
				t.Fatal(err)
			}
			sid := created.ID
			defer func() { // only our exact S; VM teardown remains the final cleanup
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				_ = manager.ServiceRemove(cleanup, sid)
			}()
			var before []swarm.Task
			ready := false
			for deadline := time.Now().Add(45 * time.Second); time.Now().Before(deadline); {
				before, err = manager.TaskList(ctx, swarm.TaskListOptions{Filters: filters.NewArgs(filters.Arg("service", sid))})
				ready = err == nil && len(before) == 2
				for _, task := range before {
					ready = ready && task.ServiceID == sid && task.NodeID == location.id && task.Status.State == swarm.TaskStateRunning && deletionContainerID(task) != ""
				}
				if ready {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if !ready {
				b, _ := json.Marshal(before)
				t.Fatalf("fixture not running: %s; %v", b, err)
			}
			tracePath := filepath.Join(s, "results", m.Engine+"-"+location.name+".jsonl")
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
			emit := func(e nativeR1Event) {
				if err := enc.Encode(e); err != nil {
					t.Fatal(err)
				}
			}
			known := map[string]swarm.Task{}
			for _, task := range before {
				known[task.ID] = task
				c, err := location.cli.ContainerInspect(ctx, deletionContainerID(task))
				if err != nil || c.State == nil || !c.State.Running {
					t.Fatalf("independent pre-DELETE oracle: %v", err)
				}
				b, _ := json.Marshal(c.State)
				emit(nativeR1Event{At: time.Now(), Kind: "oracle-before", TaskID: task.ID, Body: string(b)})
			}
			captured, deleted := false, false
			proofs := map[string]time.Time{}
			var firstDelete time.Time
			// This tap never substitutes an Engine response or waits for exit.
			// It samples the independent execution daemon once on native proof,
			// then returns identical bytes to the unchanged production helper.
			pathOf := func(req *http.Request) string {
				path := req.URL.Path
				parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
				if len(parts) == 2 && strings.HasPrefix(parts[0], "v1.") {
					path = "/" + parts[1]
				}
				return path
			}
			beforeRequest := func(req *http.Request) error {
				path := pathOf(req)
				allowed := path == "/_ping" && (req.Method == "HEAD" || req.Method == "GET") || path == "/services/"+sid && req.Method == "GET"
				if path == "/tasks" && req.Method == "GET" && !deleted {
					var f map[string]map[string]bool
					allowed = json.Unmarshal([]byte(req.URL.Query().Get("filters")), &f) == nil && len(f) == 1 && len(f["service"]) == 1 && f["service"][sid]
				}
				if path == "/services/"+sid && req.Method == "DELETE" && captured && !deleted {
					allowed, deleted, firstDelete = true, true, time.Now()
				}
				if strings.HasPrefix(path, "/tasks/") && req.Method == "GET" {
					_, allowed = known[strings.TrimPrefix(path, "/tasks/")]
				}
				if !allowed {
					return fmt.Errorf("R1 blocked helper route before network: %s %s", req.Method, req.URL.RequestURI())
				}
				emit(nativeR1Event{At: time.Now(), Kind: "helper-request", Method: req.Method, Path: req.URL.RequestURI()})
				return nil
			}
			tap := func(req *http.Request, resp *http.Response, body []byte, received time.Time) error {
				path := pathOf(req)
				emit(nativeR1Event{At: received, Kind: "helper-http", Method: req.Method, Path: req.URL.RequestURI(), Status: resp.StatusCode, Body: string(body)})
				switch {
				case path == "/_ping" && (req.Method == "HEAD" || req.Method == "GET"):
					return nil
				case path == "/services/"+sid && req.Method == "GET":
					return nil
				case path == "/tasks" && req.Method == "GET" && !deleted:
					var f map[string]map[string]bool
					if json.Unmarshal([]byte(req.URL.Query().Get("filters")), &f) != nil || len(f) != 1 || len(f["service"]) != 1 || !f["service"][sid] {
						return fmt.Errorf("unexpected capture filters")
					}
					var tasks []swarm.Task
					if resp.StatusCode != 200 || json.Unmarshal(body, &tasks) != nil || !reflect.DeepEqual(nativeR1IDs(tasks), nativeR1IDs(before)) {
						return fmt.Errorf("capture differs from the complete running fixture")
					}
					captured = true
					return nil
				case path == "/services/"+sid && req.Method == "DELETE":
					return nil
				case strings.HasPrefix(path, "/tasks/") && req.Method == "GET":
					id := strings.TrimPrefix(path, "/tasks/")
					initial, ok := known[id]
					if !ok {
						return fmt.Errorf("query outside T")
					}
					var task swarm.Task
					if resp.StatusCode != 200 {
						return nil
					} // helper must diagnose natively
					if err := json.Unmarshal(body, &task); err != nil {
						return err
					}
					if task.Status.State != swarm.TaskStateShutdown && task.Status.State != swarm.TaskStateComplete {
						return nil
					}
					if !deleted || task.ID != id || task.ServiceID != sid || task.NodeID != initial.NodeID || deletionContainerID(task) != deletionContainerID(initial) || task.Version.Index < initial.Version.Index {
						return fmt.Errorf("incoherent native proof")
					}
					start := time.Now()
					c, err := location.cli.ContainerInspect(req.Context(), deletionContainerID(initial))
					if err != nil {
						return fmt.Errorf("independent exit oracle: %w", err)
					}
					b, _ := json.Marshal(c.State)
					emit(nativeR1Event{At: start, End: time.Now(), Kind: "oracle-proof", TaskID: id, Body: string(b)})
					if c.State == nil || c.State.Running || c.State.ExitCode != 0 || c.State.FinishedAt == "" {
						return fmt.Errorf("native proof preceded independent successful exit")
					}
					logs, err := location.cli.ContainerLogs(req.Context(), c.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
					if err != nil {
						return err
					}
					var stdout, stderr bytes.Buffer
					_, copyErr := stdcopy.StdCopy(&stdout, &stderr, logs)
					_ = logs.Close()
					text := stdout.String() + stderr.String()
					emit(nativeR1Event{At: time.Now(), Kind: "oracle-logs", TaskID: id, Body: text})
					if copyErr != nil || !strings.Contains(text, "R1_READY\nR1_TERM\nR1_EXIT\n") {
						return fmt.Errorf("missing delayed TERM/exit oracle: %q (%v)", text, copyErr)
					}
					proofs[id] = received
					return nil
				}
				return fmt.Errorf("R1 forbidden helper route %s %s", req.Method, req.URL.RequestURI())
			}
			helper := nativeR1Client(t, m.Manager.Endpoint, beforeRequest, tap)
			helperCtx, stop := context.WithTimeout(ctx, 60*time.Second)
			defer stop()
			started := time.Now()
			err = deleteService(helperCtx, sid, helper)
			returned := time.Now()
			message := ""
			if err != nil {
				message = err.Error()
			}
			emit(nativeR1Event{At: returned, Kind: "helper-return", Error: message})
			t.Logf("engine=%s location=%s S=%s T=%v negotiatedAPI=%s elapsed=%s delete-request-to-return=%s proofs=%d trace=%s error=%v", m.Engine, location.name, sid, nativeR1IDs(before), helper.ClientVersion(), returned.Sub(started), returned.Sub(firstDelete), len(proofs), tracePath, err)
			if err != nil {
				t.Fatal(err)
			}
			if !deleted || len(proofs) != len(before) {
				t.Fatal("success without native/independent proof for all of T")
			}
			for _, at := range proofs {
				if at.After(returned) {
					t.Fatal("success before native task evidence")
				}
			}
			if returned.Sub(firstDelete) < 4*time.Second {
				t.Fatal("did not exercise delayed exit after direct DELETE")
			}
			if _, _, err := manager.ServiceInspectWithRaw(ctx, sid, swarm.ServiceInspectOptions{}); !serviceDeletionNotFound(err, "service", sid) {
				t.Fatalf("S absence not independently confirmed: %v", err)
			}
		}) {
			return
		} // a counterexample stops R1 rather than widening the lab
	}
}
