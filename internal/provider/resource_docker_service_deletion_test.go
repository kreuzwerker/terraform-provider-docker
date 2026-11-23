package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Every client in this file has an explicit, non-network transport. No FromEnv,
// default transport, existing socket, external command or acceptance helper.
type deletionRoundTripper func(*http.Request) (*http.Response, error)

func (f deletionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func deletionTestClient(t *testing.T, f deletionRoundTripper) *client.Client {
	t.Helper()
	cli, err := client.NewClientWithOpts(client.WithHost("http://synthetic.invalid:2375"), client.WithVersion("1.51"), client.WithHTTPClient(&http.Client{Transport: f}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func deletionJSON(code int, value interface{}) *http.Response {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}
}

func deletionAPIError(code int, message string) *http.Response {
	return deletionJSON(code, map[string]string{"message": message})
}

func deletionPath(r *http.Request) string { return strings.TrimPrefix(r.URL.Path, "/v1.51") }

func deletionTask(id string, state swarm.TaskState, grace *time.Duration) swarm.Task {
	return swarm.Task{ID: id, ServiceID: "service-original", Meta: swarm.Meta{Version: swarm.Version{Index: 1}},
		Spec:   swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{StopGracePeriod: grace}},
		Status: swarm.TaskStatus{State: state}, DesiredState: swarm.TaskStateShutdown}
}

func deletionDuration(d time.Duration) *time.Duration { return &d }

func deletionService(id string) swarm.Service {
	return swarm.Service{ID: id, Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "same-name"},
		TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: "synthetic:tag"}},
		Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{}}, EndpointSpec: &swarm.EndpointSpec{Mode: swarm.ResolutionModeVIP}}}
}

type deletionFakeClock struct {
	now   time.Time
	waits []time.Duration
	hook  func()
}

func (c *deletionFakeClock) clock() serviceDeletionClock {
	return serviceDeletionClock{now: func() time.Time { return c.now }, wait: func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		c.waits = append(c.waits, d)
		c.now = c.now.Add(d)
		if c.hook != nil {
			c.hook()
		}
		return ctx.Err()
	}}
}

func deletionContext(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func TestDockerServiceDeletionEvidence(t *testing.T) {
	states := []swarm.TaskState{swarm.TaskStateNew, swarm.TaskStateAllocated, swarm.TaskStatePending, swarm.TaskStateAssigned,
		swarm.TaskStateAccepted, swarm.TaskStatePreparing, swarm.TaskStateReady, swarm.TaskStateStarting, swarm.TaskStateRunning,
		swarm.TaskStateComplete, swarm.TaskStateShutdown, swarm.TaskStateFailed, swarm.TaskStateRejected, swarm.TaskStateOrphaned, swarm.TaskStateRemove, "future", ""}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			task := deletionTask("task-a", state, nil) // empty ContainerID does NOT remove the task
			task.Status.ContainerStatus = &swarm.ContainerStatus{ExitCode: 0}
			var record serviceDeletionTask
			err := record.observe(task, task.ServiceID, task.ID)
			wantProof := state == swarm.TaskStateComplete || state == swarm.TaskStateShutdown || state == swarm.TaskStateRejected
			if (record.proof != "") != wantProof || (err != nil) != (state == "future" || state == "") {
				t.Fatalf("state=%s proof=%q err=%v", state, record.proof, err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		edit func(*swarm.Task)
	}{
		{"empty-id", func(task *swarm.Task) { task.ID = "" }},
		{"other-task", func(task *swarm.Task) { task.ID = "other" }},
		{"other-service", func(task *swarm.Task) { task.ServiceID = "other" }},
		{"no-version", func(task *swarm.Task) { task.Version.Index = 0 }},
		{"bad-desired-state", func(task *swarm.Task) { task.DesiredState = "" }},
		{"negative-grace", func(task *swarm.Task) { task.Spec.ContainerSpec.StopGracePeriod = deletionDuration(-time.Second) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := deletionTask("task-a", swarm.TaskStateShutdown, nil)
			tc.edit(&task)
			var record serviceDeletionTask
			if err := record.observe(task, "service-original", "task-a"); err == nil || record.proof != "" {
				t.Fatalf("accepted invalid observation: %+v", record)
			}
		})
	}
	t.Run("contextual-history", func(t *testing.T) {
		for _, first := range []swarm.TaskState{swarm.TaskStatePending, swarm.TaskStateStarting, swarm.TaskStateRunning} {
			var record serviceDeletionTask
			task := deletionTask("task-a", first, nil)
			if err := record.observe(task, task.ServiceID, task.ID); err != nil {
				t.Fatal(err)
			}
			task.Version.Index++
			task.Status.State = swarm.TaskStateRejected
			err := record.observe(task, task.ServiceID, task.ID)
			if (err == nil) != (first == swarm.TaskStatePending) {
				t.Fatalf("%s -> REJECTED: %v", first, err)
			}
		}
		// SHUTDOWN supports both assigned worker and unassigned scheduler routes.
		for _, assigned := range []bool{false, true} {
			var record serviceDeletionTask
			task := deletionTask("task-a", swarm.TaskStatePending, nil)
			if assigned {
				task.NodeID = "worker"
				task.Status.State = swarm.TaskStateRunning
			}
			if err := record.observe(task, task.ServiceID, task.ID); err != nil {
				t.Fatal(err)
			}
			task.Version.Index++
			task.Status.State = swarm.TaskStateShutdown
			if err := record.observe(task, task.ServiceID, task.ID); err != nil || record.proof == "" {
				t.Fatalf("SHUTDOWN assigned=%t: %v", assigned, err)
			}
		}
	})
	t.Run("versions-and-immutable-identity", func(t *testing.T) {
		var record serviceDeletionTask
		task := deletionTask("task-a", swarm.TaskStateRunning, nil)
		task.Version.Index = 5
		task.NodeID = "worker"
		if err := record.observe(task, task.ServiceID, task.ID); err != nil {
			t.Fatal(err)
		}
		stale := task
		stale.Version.Index = 4
		stale.Status.State = swarm.TaskStatePending
		if err := record.observe(stale, task.ServiceID, task.ID); err != nil || record.last.Version.Index != 5 || record.proof != "" {
			t.Fatalf("stale snapshot replaced current: %+v, %v", record, err)
		}
		stale.Status.State = swarm.TaskStateRejected
		if err := record.observe(stale, task.ServiceID, task.ID); err == nil {
			t.Fatal("older REJECTED contradicting execution was ignored")
		}
		conflict := task
		conflict.Status.State = swarm.TaskStateShutdown
		if record.observe(conflict, task.ServiceID, task.ID) == nil {
			t.Fatal("same version contradiction accepted")
		}
		conflict.Version.Index++
		conflict.NodeID = "different-worker"
		if record.observe(conflict, task.ServiceID, task.ID) == nil {
			t.Fatal("assignment change accepted")
		}
		// Optional metadata disappearing is not itself contradictory, nor
		// is ContainerID required to admit SHUTDOWN's native disjunction.
		conflict.NodeID = ""
		if err := record.observe(conflict, task.ServiceID, task.ID); err != nil || record.proof == "" {
			t.Fatalf("missing optional metadata rejected: %v", err)
		}
	})
	t.Run("out-of-order-contradictions-invalidate", func(t *testing.T) {
		for _, earlier := range []swarm.TaskState{swarm.TaskStateComplete, swarm.TaskStateShutdown, swarm.TaskStateRejected} {
			var record serviceDeletionTask
			newer := deletionTask("a", swarm.TaskStateRunning, nil)
			newer.Version.Index = 5
			if err := record.observe(newer, newer.ServiceID, newer.ID); err != nil {
				t.Fatal(err)
			}
			older := deletionTask("a", earlier, nil)
			if err := record.observe(older, older.ServiceID, older.ID); err == nil || record.proof != "" {
				t.Fatalf("older %s ignored: %v", earlier, err)
			}
		}
		var record serviceDeletionTask
		task := deletionTask("a", swarm.TaskStateComplete, nil)
		if err := record.observe(task, task.ServiceID, task.ID); err != nil {
			t.Fatal(err)
		}
		task.Version.Index++
		task.Status.State = swarm.TaskStateRunning
		if err := record.observe(task, task.ServiceID, task.ID); err == nil || record.proof != "" {
			t.Fatalf("proof not invalidated: %v", err)
		}
	})
	t.Run("windows", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			graces []*time.Duration
			want   time.Duration
			known  bool
		}{
			{"R-empty", nil, 30 * time.Second, true},
			{"zero", []*time.Duration{deletionDuration(0)}, 30 * time.Second, true},
			{"fraction-subsecond", []*time.Duration{deletionDuration(500 * time.Millisecond)}, 30 * time.Second, true},
			{"fraction", []*time.Duration{deletionDuration(2500 * time.Millisecond)}, 32 * time.Second, true},
			{"rollout-max-not-sum", []*time.Duration{deletionDuration(7 * time.Second), deletionDuration(12 * time.Second)}, 42 * time.Second, true},
			{"nil", []*time.Duration{nil}, 0, false},
			{"one-nil", []*time.Duration{deletionDuration(time.Second), nil}, 0, false},
			{"overflow-saturates", []*time.Duration{deletionDuration(time.Duration(1<<63 - 1))}, time.Duration(1<<63 - 1), true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var tasks []*serviceDeletionTask
				for _, grace := range tc.graces {
					tasks = append(tasks, &serviceDeletionTask{last: deletionTask("a", swarm.TaskStateRunning, grace)})
				}
				got, known := serviceDeletionWindow(tasks)
				if got != tc.want || known != tc.known {
					t.Fatalf("window=%s known=%t", got, known)
				}
			})
		}
		if d, known := serviceDeletionWindow([]*serviceDeletionTask{{proof: "COMPLETE", last: deletionTask("a", swarm.TaskStateComplete, nil)}}); d != 30*time.Second || !known {
			t.Fatalf("satisfied nil task included in R: %s, %t", d, known)
		}
		missingSpec := deletionTask("a", swarm.TaskStatePending, nil)
		missingSpec.Spec.ContainerSpec = nil
		if _, known := serviceDeletionWindow([]*serviceDeletionTask{{last: missingSpec}}); known {
			t.Fatal("unreconstructible grace received an invented default")
		}
	})
}

// Scripted responses assert exact routes/order. A forbidden route cannot dial.
type deletionStep struct {
	method, path string
	response     *http.Response
	err          error
	check        func(*http.Request)
}

func deletionScript(t *testing.T, steps []deletionStep) *client.Client {
	t.Helper()
	var mu sync.Mutex
	next := 0
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		if next != len(steps) {
			t.Errorf("consumed %d/%d HTTP steps", next, len(steps))
		}
	})
	return deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		if next >= len(steps) {
			t.Errorf("unexpected HTTP %s %s", r.Method, r.URL)
			return deletionAPIError(403, "unexpected request"), nil
		}
		s := steps[next]
		next++
		if r.Method != s.method || deletionPath(r) != s.path {
			t.Errorf("HTTP step %d: got %s %s, want %s %s", next, r.Method, deletionPath(r), s.method, s.path)
		}
		if s.check != nil {
			s.check(r)
		}
		if s.response != nil {
			s.response.Request = r
		}
		return s.response, s.err
	})
}

func deletionCaptureSteps(tasks []swarm.Task) []deletionStep {
	return []deletionStep{
		{method: "GET", path: "/services/service-original", response: deletionJSON(200, deletionService("service-original"))},
		{method: "GET", path: "/tasks", response: deletionJSON(200, tasks)},
	}
}

func TestDockerServiceDeletionProtocol(t *testing.T) {
	deletionProtocolEdges(t)
	t.Run("fixed-T-empty-container-and-positive-cache", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		a := deletionTask("a", swarm.TaskStatePending, deletionDuration(time.Second))
		b := deletionTask("b", swarm.TaskStateComplete, nil) // proof already captured; never re-read after GC
		c := deletionTask("c", swarm.TaskStateFailed, deletionDuration(3*time.Second))
		c.NodeID, c.Slot = "worker-2", 3
		c.Status.ContainerStatus = &swarm.ContainerStatus{ContainerID: "container-on-worker-2"}
		steps := deletionCaptureSteps([]swarm.Task{a, b, c})
		steps[1].check = func(r *http.Request) {
			var filters map[string]map[string]bool
			if err := json.Unmarshal([]byte(r.URL.Query().Get("filters")), &filters); err != nil || !reflect.DeepEqual(filters, map[string]map[string]bool{"service": {"service-original": true}}) {
				t.Errorf("capture filter: %s (%v)", r.URL.RawQuery, err)
			}
		}
		steps = append(steps,
			deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, a)},
			deletionStep{method: "GET", path: "/tasks/c", response: deletionJSON(200, c)},
			deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)},
		)
		a.Version.Index++
		a.Status.State, a.NodeID = swarm.TaskStateRunning, "worker-1"
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, a)})
		c.Version.Index++
		c.Status.State = swarm.TaskStateShutdown
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/c", response: deletionJSON(200, c)})
		a.Version.Index++
		a.Status.State = swarm.TaskStateShutdown
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, a)})
		// Any post-delete TaskList (even empty/new tasks) or container/worker API
		// would be an unexpected step. Only the captured IDs can be observed.
		if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), clock.clock()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("lost-delete-reconciliation", func(t *testing.T) {
		for _, tc := range []struct {
			name           string
			deleteResponse *http.Response
			deleteErr      error
			present        bool
		}{
			{"lost-absent", nil, io.ErrUnexpectedEOF, false},
			{"lost-present", nil, io.ErrUnexpectedEOF, true},
			{"404-absent", deletionAPIError(404, "service service-original not found"), nil, false},
			{"500-absent", deletionAPIError(500, "temporary failure"), nil, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				clock := &deletionFakeClock{now: time.Now()}
				steps := deletionCaptureSteps([]swarm.Task{})
				steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", response: tc.deleteResponse, err: tc.deleteErr})
				if tc.present {
					steps = append(steps, deletionStep{method: "GET", path: "/services/service-original", response: deletionJSON(200, deletionService("service-original"))}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)})
				} else {
					steps = append(steps, deletionStep{method: "GET", path: "/services/service-original", response: deletionAPIError(404, "service service-original not found")})
				}
				if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), clock.clock()); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
	t.Run("resend-limit", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		steps := deletionCaptureSteps([]swarm.Task{})
		for i := 0; i < 3; i++ {
			steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionAPIError(500, "temporary failure")}, deletionStep{method: "GET", path: "/services/service-original", response: deletionJSON(200, deletionService("service-original"))})
		}
		err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), clock.clock())
		if err == nil || !strings.Contains(err.Error(), "resend limit") || !strings.Contains(err.Error(), "object=present") {
			t.Fatalf("resend limit: %v", err)
		}
	})
	t.Run("no-blind-resend", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		steps := deletionCaptureSteps([]swarm.Task{})
		steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", err: io.ErrUnexpectedEOF}, deletionStep{method: "GET", path: "/services/service-original", response: deletionAPIError(403, "denied")})
		err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), clock.clock())
		if err == nil || !strings.Contains(err.Error(), "object=unknown") {
			t.Fatalf("reconciliation unknown: %v", err)
		}
	})
	t.Run("capture-rejections-no-delete", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			response *http.Response
		}{
			{"401", deletionAPIError(401, "denied")}, {"403", deletionAPIError(403, "denied")},
			{"null", deletionJSON(200, nil)}, {"wrong-service", deletionJSON(200, []swarm.Task{func() swarm.Task {
				task := deletionTask("a", swarm.TaskStateRunning, nil)
				task.ServiceID = "other"
				return task
			}()})},
			{"truncated", &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`[{"ID":`))}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				steps := deletionCaptureSteps([]swarm.Task{})
				steps[1].response = tc.response
				err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), (&deletionFakeClock{now: time.Now()}).clock())
				if err == nil {
					t.Fatal("invalid capture accepted")
				}
			})
		}
	})
	t.Run("404-is-not-route-or-exit-proof", func(t *testing.T) {
		for _, message := range []string{"404 page not found", "<html>not found</html>", "service other not found"} {
			t.Run(message, func(t *testing.T) {
				cli := deletionScript(t, []deletionStep{{method: "GET", path: "/services/service-original", response: deletionAPIError(404, message)}})
				if err := deleteService(deletionContext(t, time.Second), "service-original", cli); err == nil || strings.Contains(err.Error(), "object=removed") {
					t.Fatalf("route error treated as absence: %v", err)
				}
			})
		}
	})
	t.Run("gc-without-proof-and-deadline", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		task := deletionTask("a", swarm.TaskStatePending, deletionDuration(0))
		deleted, reads := false, 0
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			switch r.Method + " " + deletionPath(r) {
			case "GET /services/service-original":
				return deletionJSON(200, deletionService("service-original")), nil
			case "GET /tasks":
				return deletionJSON(200, []swarm.Task{task}), nil
			case "DELETE /services/service-original":
				deleted = true
				return deletionJSON(200, nil), nil
			case "GET /tasks/a":
				reads++
				return deletionAPIError(404, "task a not found"), nil
			default:
				t.Errorf("forbidden request: %s %s", r.Method, r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", cli, clock.clock())
		if err == nil || !errors.Is(err, context.DeadlineExceeded) || !deleted || reads < 2 || !strings.Contains(err.Error(), "object=removed") || !strings.Contains(err.Error(), "captured=1, satisfied=0") {
			t.Fatalf("GC: deleted=%t reads=%d err=%v", deleted, reads, err)
		}
	})
	t.Run("post-delete-transient-recovery", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		task := deletionTask("a", swarm.TaskStateRunning, nil)
		steps := deletionCaptureSteps([]swarm.Task{task})
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)}, deletionStep{method: "GET", path: "/tasks/a", response: deletionAPIError(500, "temporarily unavailable")})
		task.Version.Index++
		task.Status.State = swarm.TaskStateComplete
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)})
		if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), clock.clock()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("window-fixed-from-first-delete", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			grace       *time.Duration
			sdk, window time.Duration
		}{
			{"known", deletionDuration(2500 * time.Millisecond), time.Minute, 32 * time.Second},
			{"SDK-shorter", deletionDuration(time.Minute), 12 * time.Second, 12 * time.Second},
			{"nil-SDK-only", nil, 2 * time.Minute, 2 * time.Minute},
		} {
			t.Run(tc.name, func(t *testing.T) {
				clock := &deletionFakeClock{now: time.Now()}
				ctx, cancel := context.WithDeadline(context.Background(), clock.now.Add(tc.sdk))
				defer cancel()
				task := deletionTask("a", swarm.TaskStateRunning, tc.grace)
				deadline := clock.now.Add(tc.window)
				checkDeadline := func(r *http.Request) {
					got, _ := r.Context().Deadline()
					if !got.Equal(deadline) {
						t.Errorf("deadline=%v want=%v", got, deadline)
					}
				}
				steps := deletionCaptureSteps([]swarm.Task{task})
				steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)}, deletionStep{method: "DELETE", path: "/services/service-original", err: io.ErrUnexpectedEOF, check: func(r *http.Request) { checkDeadline(r); clock.now = clock.now.Add(time.Second) }}, deletionStep{method: "GET", path: "/services/service-original", response: deletionJSON(200, deletionService("service-original")), check: checkDeadline}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil), check: checkDeadline})
				task.Version.Index++
				task.Status.State = swarm.TaskStateShutdown
				steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task), check: checkDeadline})
				if err := deleteServiceWithClock(ctx, "service-original", deletionScript(t, steps), clock.clock()); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
	t.Run("nil-exhaustion-no-added-budget", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		ctx, cancel := context.WithDeadline(context.Background(), clock.now.Add(75*time.Second))
		defer cancel()
		start := clock.now
		task := deletionTask("a", swarm.TaskStateFailed, nil)
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			switch r.Method + " " + deletionPath(r) {
			case "GET /services/service-original":
				return deletionJSON(200, deletionService("service-original")), nil
			case "GET /tasks":
				return deletionJSON(200, []swarm.Task{task}), nil
			case "GET /tasks/a":
				return deletionJSON(200, task), nil
			case "DELETE /services/service-original":
				return deletionJSON(200, nil), nil
			default:
				t.Errorf("unexpected %s", r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		err := deleteServiceWithClock(ctx, "service-original", cli, clock.clock())
		if err == nil || !strings.Contains(err.Error(), "only the received SDK context deadline") || clock.now.Sub(start) != 75*time.Second {
			t.Fatalf("nil exception: elapsed=%s err=%v", clock.now.Sub(start), err)
		}
	})
	t.Run("unbounded-or-empty-identity-refused-without-requests", func(t *testing.T) {
		if err := deleteService(context.Background(), "service-original", deletionScript(t, nil)); err == nil {
			t.Fatal("unbounded context accepted")
		}
		if err := deleteService(deletionContext(t, time.Second), "", deletionScript(t, nil)); err == nil {
			t.Fatal("empty ID accepted")
		}
	})
	t.Run("cancel-before-delete", func(t *testing.T) {
		ctx, cancel := context.WithCancel(deletionContext(t, time.Minute))
		cancel()
		cli := deletionScript(t, nil)
		if err := deleteService(ctx, "service-original", cli); !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	})
	t.Run("cancel-after-delete-and-slow-reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(deletionContext(t, time.Minute))
		defer cancel()
		task := deletionTask("a", swarm.TaskStateRunning, nil)
		steps := deletionCaptureSteps([]swarm.Task{task})
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)})
		base := deletionScript(t, steps)
		post := false
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			if post {
				cancel()
				<-r.Context().Done()
				return nil, r.Context().Err()
			}
			response, err := base.HTTPClient().Transport.RoundTrip(r)
			if r.Method == "DELETE" {
				post = true
			}
			return response, err
		})
		if err := deleteServiceWithClock(ctx, "service-original", cli, (&deletionFakeClock{now: time.Now()}).clock()); !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	})
	t.Run("large-queue-fair-bounded-rate", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		tasks := make([]swarm.Task, 120)
		for i := range tasks {
			tasks[i] = deletionTask(fmt.Sprintf("task-%03d", i), swarm.TaskStatePending, deletionDuration(0))
		}
		post, before, after := false, 0, 0
		var last time.Time
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			switch {
			case deletionPath(r) == "/services/service-original" && r.Method == "GET":
				return deletionJSON(200, deletionService("service-original")), nil
			case deletionPath(r) == "/tasks":
				return deletionJSON(200, tasks), nil
			case r.Method == "DELETE":
				post = true
				return deletionJSON(200, nil), nil
			case strings.HasPrefix(deletionPath(r), "/tasks/task-"):
				if !last.IsZero() && clock.now.Sub(last) < 20*time.Millisecond {
					t.Error("rate exceeded")
				}
				last = clock.now
				index := before
				if post {
					index = after
					after++
				} else {
					before++
				}
				if index >= len(tasks) {
					t.Fatal("duplicate or unbounded read")
				}
				task := tasks[index]
				if deletionPath(r) != "/tasks/"+task.ID {
					t.Errorf("unfair queue: %s", r.URL)
				}
				if post {
					task.Version.Index++
					task.Status.State = swarm.TaskStateShutdown
				}
				return deletionJSON(200, task), nil
			default:
				t.Errorf("forbidden %s", r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", cli, clock.clock()); err != nil || before != len(tasks) || after != len(tasks) {
			t.Fatalf("before=%d after=%d: %v", before, after, err)
		}
	})
}

func deletionProtocolEdges(t *testing.T) {
	t.Run("capture-deduplicates-only-consistent-identities", func(t *testing.T) {
		for _, contradictory := range []bool{false, true} {
			t.Run(fmt.Sprintf("contradictory=%t", contradictory), func(t *testing.T) {
				a := deletionTask("a", swarm.TaskStateComplete, nil)
				b := a
				if contradictory {
					b.Version.Index++
					b.Status.State = swarm.TaskStateRunning
				}
				steps := deletionCaptureSteps([]swarm.Task{a, b})
				if !contradictory {
					steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)})
				}
				err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), (&deletionFakeClock{now: time.Now()}).clock())
				if (err != nil) != contradictory {
					t.Fatalf("%v", err)
				}
			})
		}
	})
	t.Run("permanent-preflight-and-delete-errors", func(t *testing.T) {
		for _, phase := range []string{"inspect", "pre-task", "delete"} {
			for _, code := range []int{401, 403, 404} {
				t.Run(fmt.Sprintf("%s-%d", phase, code), func(t *testing.T) {
					task := deletionTask("a", swarm.TaskStateRunning, nil)
					response := deletionAPIError(code, "route unavailable")
					if code == 404 {
						response = &http.Response{StatusCode: 404, Header: http.Header{"Content-Type": {"text/html"}}, Body: io.NopCloser(strings.NewReader("<html>route unavailable</html>"))}
					}
					steps := deletionCaptureSteps([]swarm.Task{task})
					switch phase {
					case "inspect":
						steps = []deletionStep{{method: "GET", path: "/services/service-original", response: response}}
					case "pre-task":
						steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: response})
					case "delete":
						steps = append(steps,
							deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)},
							deletionStep{method: "DELETE", path: "/services/service-original", response: response},
							deletionStep{method: "GET", path: "/services/service-original", response: deletionJSON(200, deletionService("service-original"))})
					}
					if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), (&deletionFakeClock{now: time.Now()}).clock()); err == nil {
						t.Fatal("permanent/routing error accepted")
					}
				})
			}
		}
	})
	t.Run("preobservation-transient-queue-is-fair", func(t *testing.T) {
		a := deletionTask("a", swarm.TaskStatePending, deletionDuration(0))
		b := deletionTask("b", swarm.TaskStatePending, deletionDuration(0))
		steps := deletionCaptureSteps([]swarm.Task{a, b})
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionAPIError(500, "temporary")}, deletionStep{method: "GET", path: "/tasks/b", response: deletionJSON(200, b)}, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, a)}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)})
		a.Version.Index++
		a.Status.State = swarm.TaskStateShutdown
		b.Version.Index++
		b.Status.State = swarm.TaskStateShutdown
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, a)}, deletionStep{method: "GET", path: "/tasks/b", response: deletionJSON(200, b)})
		if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), (&deletionFakeClock{now: time.Now()}).clock()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("capture-retry-and-pre-task-404-recovery", func(t *testing.T) {
		task := deletionTask("a", swarm.TaskStatePending, nil)
		steps := deletionCaptureSteps([]swarm.Task{task})
		steps = append(steps[:1], append([]deletionStep{{method: "GET", path: "/tasks", response: deletionAPIError(500, "temporary")}}, steps[1:]...)...)
		steps = append(steps,
			deletionStep{method: "GET", path: "/tasks/a", response: deletionAPIError(404, "task a not found")},
			deletionStep{method: "DELETE", path: "/services/service-original", err: io.ErrUnexpectedEOF},
			deletionStep{method: "GET", path: "/services/service-original", response: deletionAPIError(500, "temporary")},
			deletionStep{method: "GET", path: "/services/service-original", response: deletionAPIError(404, "service service-original not found")},
		)
		task.Version.Index++
		task.Status.State = swarm.TaskStateShutdown
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)})
		if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", deletionScript(t, steps), (&deletionFakeClock{now: time.Now()}).clock()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("positive-after-delete-retained-and-new-task-outside-T", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		a := deletionTask("a", swarm.TaskStateRunning, deletionDuration(0))
		b := deletionTask("b", swarm.TaskStatePending, deletionDuration(0))
		post, aReads, bReads, lists := false, 0, 0, 0
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			switch r.Method + " " + deletionPath(r) {
			case "GET /services/service-original":
				return deletionJSON(200, deletionService("service-original")), nil
			case "GET /tasks":
				lists++
				if post {
					t.Error("post-delete list forbidden; a new task is outside T")
					return deletionJSON(200, []swarm.Task{deletionTask("new", swarm.TaskStateRunning, nil)}), nil
				}
				return deletionJSON(200, []swarm.Task{a, b}), nil
			case "DELETE /services/service-original":
				post = true
				return deletionJSON(200, nil), nil
			case "GET /tasks/a":
				aReads++
				if post {
					if aReads > 2 {
						t.Error("positive proof re-read after GC")
						return deletionAPIError(404, "task a not found"), nil
					}
					a.Version.Index++
					a.Status.State = swarm.TaskStateComplete
				}
				return deletionJSON(200, a), nil
			case "GET /tasks/b":
				bReads++
				if post && bReads >= 4 {
					b.Version.Index++
					b.Status.State = swarm.TaskStateShutdown
				}
				return deletionJSON(200, b), nil
			default:
				t.Errorf("forbidden request %s", r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		if err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", cli, clock.clock()); err != nil || lists != 1 || aReads != 2 || bReads != 4 {
			t.Fatalf("a=%d b=%d lists=%d: %v", aReads, bReads, lists, err)
		}
	})
	t.Run("R-frozen-after-preobservation-max-and-empty", func(t *testing.T) {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("empty=%t", empty), func(t *testing.T) {
				clock := &deletionFakeClock{now: time.Now()}
				start := clock.now
				a := deletionTask("nil-already-complete", swarm.TaskStateComplete, nil)
				b := deletionTask("old-spec", swarm.TaskStateRunning, deletionDuration(7*time.Second))
				c := deletionTask("new-spec", swarm.TaskStateRunning, deletionDuration(12500*time.Millisecond))
				tasks := []swarm.Task{a, b, c}
				window := 42 * time.Second
				if empty {
					tasks = []swarm.Task{}
					window = 30 * time.Second
				}
				steps := deletionCaptureSteps(tasks)
				// Capture time is NOT charged to the first-DELETE window.
				steps[1].check = func(*http.Request) { clock.now = clock.now.Add(9 * time.Second) }
				if !empty {
					steps = append(steps, deletionStep{method: "GET", path: "/tasks/old-spec", response: deletionJSON(200, b)}, deletionStep{method: "GET", path: "/tasks/new-spec", response: deletionJSON(200, c)})
				}
				steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil), check: func(r *http.Request) {
					deadline, _ := r.Context().Deadline()
					want := clock.now.Add(window)
					if !deadline.Equal(want) {
						t.Errorf("deadline=%s want=%s", deadline, want)
					}
				}})
				if !empty {
					b.Version.Index++
					b.Status.State = swarm.TaskStateShutdown
					c.Version.Index++
					c.Status.State = swarm.TaskStateShutdown
					steps = append(steps, deletionStep{method: "GET", path: "/tasks/old-spec", response: deletionJSON(200, b)}, deletionStep{method: "GET", path: "/tasks/new-spec", response: deletionJSON(200, c)})
				}
				if err := deleteServiceWithClock(deletionContext(t, 2*time.Minute), "service-original", deletionScript(t, steps), clock.clock()); err != nil {
					t.Fatal(err)
				}
				if clock.now.Sub(start) > 10*time.Second {
					t.Fatal("slept grace/window instead of returning on proof")
				}
			})
		}
	})
	t.Run("nil-budget-not-recomputed-after-partial-proof", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		ctx, cancel := context.WithDeadline(context.Background(), clock.now.Add(75*time.Second))
		defer cancel()
		deadline, _ := ctx.Deadline()
		a := deletionTask("nil", swarm.TaskStateRunning, nil)
		b := deletionTask("known", swarm.TaskStateFailed, deletionDuration(0))
		post, nilReads, sends := false, 0, 0
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			if post {
				got, _ := r.Context().Deadline()
				if !got.Equal(deadline) {
					t.Error("nil budget recomputed after partial proof")
				}
			}
			switch r.Method + " " + deletionPath(r) {
			case "GET /services/service-original":
				return deletionJSON(200, deletionService("service-original")), nil
			case "GET /tasks":
				return deletionJSON(200, []swarm.Task{a, b}), nil
			case "GET /tasks/nil":
				nilReads++
				if post {
					a.Version.Index++
					a.Status.State = swarm.TaskStateShutdown
				}
				return deletionJSON(200, a), nil
			case "GET /tasks/known":
				return deletionJSON(200, b), nil
			case "DELETE /services/service-original":
				post = true
				sends++
				return deletionJSON(200, nil), nil
			default:
				t.Errorf("unexpected %s", r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		err := deleteServiceWithClock(ctx, "service-original", cli, clock.clock())
		if err == nil || !errors.Is(err, context.DeadlineExceeded) || !clock.now.Equal(deadline) || nilReads != 2 || sends != 1 || !strings.Contains(err.Error(), "captured=2, satisfied=1") || !strings.Contains(err.Error(), "only the received SDK context deadline") {
			t.Fatalf("nilReads=%d sends=%d: %v", nilReads, sends, err)
		}
	})
	t.Run("cancel-during-backoff-no-resend", func(t *testing.T) {
		ctx, cancel := context.WithCancel(deletionContext(t, time.Minute))
		defer cancel()
		clock := &deletionFakeClock{now: time.Now(), hook: cancel}
		steps := deletionCaptureSteps([]swarm.Task{})
		steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", err: io.ErrUnexpectedEOF}, deletionStep{method: "GET", path: "/services/service-original", response: deletionJSON(200, deletionService("service-original"))})
		if err := deleteServiceWithClock(ctx, "service-original", deletionScript(t, steps), clock.clock()); !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	})
	t.Run("unresolved-reconciliation-exhausts-no-resend", func(t *testing.T) {
		clock := &deletionFakeClock{now: time.Now()}
		sends, inspects := 0, 0
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			switch r.Method + " " + deletionPath(r) {
			case "GET /services/service-original":
				inspects++
				if inspects == 1 {
					return deletionJSON(200, deletionService("service-original")), nil
				}
				return deletionAPIError(500, "temporary"), nil
			case "GET /tasks":
				return deletionJSON(200, []swarm.Task{}), nil
			case "DELETE /services/service-original":
				sends++
				return nil, io.ErrUnexpectedEOF
			default:
				t.Errorf("forbidden %s", r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		err := deleteServiceWithClock(deletionContext(t, time.Minute), "service-original", cli, clock.clock())
		if err == nil || !errors.Is(err, context.DeadlineExceeded) || sends != 1 || !strings.Contains(err.Error(), "object=unknown") {
			t.Fatalf("sends=%d err=%v", sends, err)
		}
	})
	t.Run("slow-reader-receives-only-SDK-deadline", func(t *testing.T) {
		ctx := deletionContext(t, 100*time.Millisecond)
		task := deletionTask("a", swarm.TaskStateRunning, nil)
		post := false
		cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
			switch r.Method + " " + deletionPath(r) {
			case "GET /services/service-original":
				return deletionJSON(200, deletionService("service-original")), nil
			case "GET /tasks":
				return deletionJSON(200, []swarm.Task{task}), nil
			case "DELETE /services/service-original":
				post = true
				return deletionJSON(200, nil), nil
			case "GET /tasks/a":
				if post {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return deletionJSON(200, task), nil
			default:
				t.Errorf("forbidden %s", r.URL)
				return deletionAPIError(403, "forbidden"), nil
			}
		})
		if err := deleteService(ctx, "service-original", cli); !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "object=removed") {
			t.Fatalf("%v", err)
		}
	})
}

func deletionResourceData(t *testing.T, grace string) *schema.ResourceData {
	t.Helper()
	container := map[string]interface{}{"image": "synthetic:tag"}
	if grace != "" {
		container["stop_grace_period"] = grace
	}
	return schema.TestResourceDataRaw(t, resourceDockerService().Schema, map[string]interface{}{"name": "same-name", "task_spec": []interface{}{map[string]interface{}{"container_spec": []interface{}{container}}}})
}

func deletionMeta(cli *client.Client) *ProviderConfig {
	config := &Config{Host: "http://synthetic.invalid:2375"}
	meta := &ProviderConfig{DefaultConfig: config, AuthConfigs: &AuthConfigs{}}
	meta.clientCache.Store(config.Hash(), cli)
	return meta
}

func deletionDiagnostics(diags diag.Diagnostics) string {
	var parts []string
	for _, d := range diags {
		parts = append(parts, d.Summary, d.Detail)
	}
	return strings.Join(parts, "\n")
}

func TestDockerServiceDeletionCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name           string
		absent, denied bool
	}{{"Delete-success", false, false}, {"Delete-initially-absent", true, false}, {"Delete-denied", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			d := deletionResourceData(t, "")
			d.SetId("service-original")
			steps := deletionCaptureSteps([]swarm.Task{})
			if tc.absent {
				steps = []deletionStep{{method: "GET", path: "/services/service-original", response: deletionAPIError(404, "service service-original not found")}}
			} else if tc.denied {
				steps[1].response = deletionAPIError(403, "denied")
			} else {
				steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)})
			}
			diags := resourceDockerServiceDelete(deletionContext(t, time.Second), d, deletionMeta(deletionScript(t, steps)))
			wantError := tc.absent || tc.denied
			if diags.HasError() != wantError || (d.Id() != "") != wantError {
				t.Fatalf("id=%q diags=%v", d.Id(), diags)
			}
		})
	}
	t.Run("Delete-post-delete-unknown-retains-ID", func(t *testing.T) {
		d := deletionResourceData(t, "0s")
		d.SetId("service-original")
		task := deletionTask("a", swarm.TaskStateRunning, nil)
		steps := deletionCaptureSteps([]swarm.Task{task})
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task)}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)}, deletionStep{method: "GET", path: "/tasks/a", response: deletionAPIError(403, "denied")})
		diags := resourceDockerServiceDelete(deletionContext(t, time.Second), d, deletionMeta(deletionScript(t, steps)))
		if !diags.HasError() || d.Id() != "service-original" || !strings.Contains(deletionDiagnostics(diags), "object=removed") {
			t.Fatalf("id=%q %v", d.Id(), diags)
		}
	})
	for _, tc := range []struct {
		name, id, returned string
		code               int
		wantID, wantState  string
		wantError          bool
	}{
		{"Read-absent-homonym-not-adopted", "service-original", "", 404, "", "removed", false},
		{"Read-denied", "service-original", "", 403, "service-original", "", true},
		{"Read-wrong-identity", "service-original", "service-replacement", 200, "service-original", "", true},
		{"Read-full-import", "service-original", "service-original", 200, "service-original", "all_fields", false},
		{"Read-short-import", "service-orig", "service-original", 200, "service-original", "all_fields", false},
		{"Read-name-import-rejected", "same-name", "service-original", 200, "same-name", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := deletionResourceData(t, "")
			d.SetId(tc.id)
			if strings.Contains(tc.name, "import") {
				states, err := resourceDockerService().Importer.StateContext(context.Background(), d, nil)
				if err != nil || len(states) != 1 || states[0].Id() != tc.id {
					t.Fatalf("passthrough import: %v, %v", states, err)
				}
			}
			response := deletionJSON(200, deletionService(tc.returned))
			if tc.code != 200 {
				message := "denied"
				if tc.code == 404 {
					message = "service " + tc.id + " not found"
				}
				response = deletionAPIError(tc.code, message)
			}
			cli := deletionScript(t, []deletionStep{{method: "GET", path: "/services/" + tc.id, response: response}})
			_, state, err := resourceDockerServiceReadRefreshFunc(deletionContext(t, time.Second), d, nil, cli)()
			if (err != nil) != tc.wantError || state != tc.wantState || d.Id() != tc.wantID {
				t.Fatalf("id=%q state=%q err=%v", d.Id(), state, err)
			}
		})
	}
	for _, outcome := range []string{"success", "capture-denied", "post-delete-denied", "cancelled"} {
		t.Run("Create-cleanup-"+outcome, func(t *testing.T) {
			d := deletionResourceData(t, "")
			_ = d.Set("converge_config", []interface{}{map[string]interface{}{"delay": "0s", "timeout": "1s"}})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			inspect, lists := 0, 0
			task := deletionTask("a", swarm.TaskStateRunning, nil)
			cli := deletionTestClient(t, func(r *http.Request) (*http.Response, error) {
				switch r.Method + " " + deletionPath(r) {
				case "GET /distribution/synthetic:tag/json":
					// Existing ServiceCreate SDK registry resolution, through the
					// same fake manager transport, never a registry or Docker daemon.
					return deletionAPIError(403, "synthetic registry resolution unavailable"), nil
				case "POST /services/create":
					return deletionJSON(201, swarm.ServiceCreateResponse{ID: "service-original"}), nil
				case "GET /services/service-original":
					inspect++
					if d.Id() != "service-original" {
						t.Error("ID was not registered before convergence")
					}
					if inspect == 1 {
						return deletionAPIError(403, "convergence-original-cause"), nil
					}
					return deletionJSON(200, deletionService("service-original")), nil
				case "GET /tasks":
					lists++
					if lists == 1 {
						if outcome == "cancelled" {
							cancel()
						}
						return deletionJSON(200, []swarm.Task{}), nil
					} // auxiliary logs
					if outcome == "capture-denied" {
						return deletionAPIError(403, "cleanup-capture-denied"), nil
					}
					if outcome == "post-delete-denied" {
						return deletionJSON(200, []swarm.Task{task}), nil
					}
					return deletionJSON(200, []swarm.Task{}), nil
				case "GET /tasks/a":
					if lists == 2 {
						lists++
						return deletionJSON(200, task), nil
					}
					return deletionAPIError(403, "cleanup-observation-denied"), nil
				case "DELETE /services/service-original":
					return deletionJSON(200, nil), nil
				default:
					t.Errorf("unexpected Create HTTP: %s %s", r.Method, r.URL)
					return deletionAPIError(403, "unexpected"), nil
				}
			})
			diags := resourceDockerServiceCreate(ctx, d, deletionMeta(cli))
			text := deletionDiagnostics(diags)
			if !diags.HasError() || !strings.Contains(text, "convergence-original-cause") {
				t.Fatalf("lost convergence error: %s", text)
			}
			if outcome == "success" {
				if d.Id() != "" {
					t.Fatalf("cleanup success id=%q", d.Id())
				}
			} else if d.Id() != "service-original" || !strings.Contains(text, "cleanup:") {
				t.Fatalf("cleanup lost ID or second cause: %s, id=%q", text, d.Id())
			}
		})
	}
}

type deletionTrackedBody struct {
	io.Reader
	closed bool
}

func (b *deletionTrackedBody) Close() error { b.closed = true; return nil }

func TestDockerServiceDeletionLogs(t *testing.T) {
	for _, outcome := range []string{"available", "empty", "denied", "inspect-error", "empty-container", "cancelled"} {
		t.Run(outcome, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			task := deletionTask("failed", swarm.TaskStateFailed, nil)
			task.Status.ContainerStatus = &swarm.ContainerStatus{ContainerID: "container-failed"}
			if outcome == "empty-container" {
				task.Status.ContainerStatus.ContainerID = ""
			}
			// >64KiB aggregate in short lines: no new total byte quota.
			payload := strings.Repeat("12345678useful-log-line\n", 6000)
			if outcome == "empty" {
				payload = ""
			}
			body := &deletionTrackedBody{Reader: strings.NewReader(payload)}
			steps := []deletionStep{{method: "GET", path: "/tasks", response: deletionJSON(200, []swarm.Task{task}), check: func(r *http.Request) {
				var filters map[string]map[string]bool
				_ = json.Unmarshal([]byte(r.URL.Query().Get("filters")), &filters)
				if !filters["service"]["service-original"] {
					t.Error("logs not looked up by original ID")
				}
			}}}
			if outcome == "inspect-error" {
				steps = append(steps, deletionStep{method: "GET", path: "/tasks/failed", response: deletionAPIError(403, "denied")})
			} else {
				steps = append(steps, deletionStep{method: "GET", path: "/tasks/failed", response: deletionJSON(200, task)})
				if outcome != "empty-container" {
					response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/vnd.docker.raw-stream"}}, Body: body}
					if outcome == "denied" {
						response = deletionAPIError(403, "logs unavailable")
					}
					step := deletionStep{method: "GET", path: "/containers/container-failed/logs", response: response, check: func(r *http.Request) {
						query := r.URL.Query()
						if query.Get("tail") != "10" || query.Get("stdout") != "1" || query.Get("stderr") != "1" {
							t.Errorf("logs options: %s", r.URL)
						}
						got, _ := r.Context().Deadline()
						want, _ := ctx.Deadline()
						if !got.Equal(want) {
							t.Error("new logs deadline introduced")
						}
					}}
					if outcome == "cancelled" {
						step.response = nil
						step.err = context.Canceled
						step.check = func(r *http.Request) { cancel(); <-r.Context().Done() }
					}
					steps = append(steps, step)
				}
			}
			logs, logsErr := failedDockerServiceLogs(ctx, "service-original", deletionScript(t, steps))
			if (logsErr != nil) != (outcome != "available" && outcome != "empty") {
				t.Fatalf("logs availability not distinguished: %v", logsErr)
			}
			if outcome == "available" {
				if !body.closed || strings.Count(logs, "useful-log-line") != 6000 {
					t.Fatalf("logs truncated or not closed: bytes=%d closed=%t", len(logs), body.closed)
				}
			} else if logs != "" {
				t.Fatalf("unexpected logs: %q", logs)
			}
			if outcome == "empty" && !body.closed {
				t.Fatal("empty log stream not closed")
			}
		})
	}
	t.Run("Create-cleanup-nil-uses-Create-remainder", func(t *testing.T) {
		ctx := deletionContext(t, 800*time.Millisecond)
		deadline, _ := ctx.Deadline()
		d := deletionResourceData(t, "")
		d.SetId("service-original")
		task := deletionTask("a", swarm.TaskStateRunning, nil)
		steps := []deletionStep{{method: "GET", path: "/tasks", response: deletionJSON(200, []swarm.Task{})}}
		steps = append(steps, deletionCaptureSteps([]swarm.Task{task})...)
		check := func(r *http.Request) {
			got, _ := r.Context().Deadline()
			if !got.Equal(deadline) {
				t.Error("cleanup extended Create context or reused converge_config")
			}
		}
		steps = append(steps, deletionStep{method: "GET", path: "/tasks/a", response: deletionJSON(200, task), check: check}, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil), check: check}, deletionStep{method: "GET", path: "/tasks/a", response: deletionAPIError(403, "observation-denied"), check: check})
		diags := cleanupFailedDockerServiceCreate(ctx, d, deletionScript(t, steps), &convergeConfig{timeout: time.Millisecond}, errors.New("original-convergence"))
		text := deletionDiagnostics(diags)
		if d.Id() != "service-original" || !strings.Contains(text, "original-convergence") || !strings.Contains(text, "only the received SDK context deadline") {
			t.Fatalf("id=%q %s", d.Id(), text)
		}
	})
	t.Run("logs-error-does-not-decide-cleanup", func(t *testing.T) {
		d := deletionResourceData(t, "")
		d.SetId("service-original")
		steps := []deletionStep{{method: "GET", path: "/tasks", response: deletionAPIError(403, "logs-unavailable")}}
		steps = append(steps, deletionCaptureSteps([]swarm.Task{})...)
		steps = append(steps, deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)})
		diags := cleanupFailedDockerServiceCreate(deletionContext(t, time.Second), d, deletionScript(t, steps), &convergeConfig{timeout: time.Second}, errors.New("timeout while waiting for state: original-details"))
		text := deletionDiagnostics(diags)
		if d.Id() != "" || !diags.HasError() || !strings.Contains(text, "original-details") || !strings.Contains(text, "logs-unavailable") || strings.Contains(text, "cleanup:") {
			t.Fatalf("logs changed B or lost cause: id=%q, %s", d.Id(), text)
		}
	})
	t.Run("both-errors-and-logs", func(t *testing.T) {
		d := deletionResourceData(t, "")
		d.SetId("service-original")
		task := deletionTask("failed", swarm.TaskStateFailed, nil)
		task.Status.ContainerStatus = &swarm.ContainerStatus{ContainerID: "container-failed"}
		steps := []deletionStep{
			{method: "GET", path: "/tasks", response: deletionJSON(200, []swarm.Task{task})},
			{method: "GET", path: "/tasks/failed", response: deletionJSON(200, task)},
			{method: "GET", path: "/containers/container-failed/logs", response: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("12345678useful diagnosis\n"))}},
			{method: "GET", path: "/services/service-original", response: deletionAPIError(403, "cleanup-denied")},
		}
		diags := cleanupFailedDockerServiceCreate(deletionContext(t, time.Second), d, deletionScript(t, steps), &convergeConfig{}, errors.New("original-convergence"))
		text := deletionDiagnostics(diags)
		for _, part := range []string{"original-convergence", "cleanup-denied", "useful diagnosis"} {
			if !strings.Contains(text, part) {
				t.Errorf("missing %s: %s", part, text)
			}
		}
		if d.Id() != "service-original" {
			t.Fatal("lost ID")
		}
	})
}

func TestDockerServiceDeletionSchema(t *testing.T) {
	t.Run("flatten-nil-does-not-invent-a-grace", func(t *testing.T) {
		flat := flattenContainerSpec(&swarm.ContainerSpec{Image: "synthetic:tag"}, deletionScript(t, nil))
		if _, exists := flat[0].(map[string]interface{})["stop_grace_period"]; exists {
			t.Fatal("flatten invented a grace")
		}
	})
	for _, grace := range []string{"", "0s", "0ms", "2.5s"} {
		t.Run("grace="+grace, func(t *testing.T) {
			d := deletionResourceData(t, grace)
			d.SetId("service-original")
			cli := deletionScript(t, nil)
			spec, err := createContainerSpec(d.Get("task_spec.0.container_spec"), cli)
			if err != nil {
				t.Fatal(err)
			}
			// SDK-synthetic omitted input may materialize an empty computed string;
			// record the existing conversion, not a claim about Core/HCL null.
			if spec.StopGracePeriod == nil {
				t.Fatal("ResourceData did not materialize existing grace conversion")
			}
			want := time.Duration(0)
			if grace != "" {
				want, _ = time.ParseDuration(grace)
			}
			if *spec.StopGracePeriod != want {
				t.Fatalf("marshal changed: %s", *spec.StopGracePeriod)
			}
			flattened := flattenContainerSpec(spec, cli)
			if got := flattened[0].(map[string]interface{})["stop_grace_period"]; got != shortDur(want) {
				t.Fatalf("flatten changed grace: %v", got)
			}
			task := deletionTask("task-a", swarm.TaskStatePending, spec.StopGracePeriod)
			steps := deletionCaptureSteps([]swarm.Task{task})
			steps = append(steps,
				deletionStep{method: "GET", path: "/tasks/task-a", response: deletionJSON(200, task)},
				deletionStep{method: "DELETE", path: "/services/service-original", response: deletionJSON(200, nil)},
			)
			task.Status.State = swarm.TaskStateShutdown
			task.Version.Index++
			steps = append(steps, deletionStep{method: "GET", path: "/tasks/task-a", response: deletionJSON(200, task)})
			if diags := resourceDockerServiceDelete(deletionContext(t, time.Second), d, deletionMeta(deletionScript(t, steps))); diags.HasError() || d.Id() != "" {
				t.Fatalf("grace bypass/error: %v id=%q", diags, d.Id())
			}
		})
	}
}
