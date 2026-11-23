package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// These records belong to one invocation, not to Terraform state. In particular,
// a subsequent refresh/retry cannot reconstruct tasks already lost to GC.
type serviceDeletionTask struct {
	last        swarm.Task
	seen        bool
	started     bool
	nodeID      string
	containerID string
	proof       string
	err         error
}

func (t *serviceDeletionTask) observe(task swarm.Task, serviceID, taskID string) (err error) {
	defer func() {
		if err != nil {
			t.proof, t.err = "", err // contradictions invalidate, unlike metadata GC
		}
	}()
	if task.ID == "" || task.ID != taskID || task.ServiceID != serviceID || task.Version.Index == 0 {
		return fmt.Errorf("invalid identity or version for task %q", taskID)
	}
	if !deletionTaskStateKnown(task.Status.State) || !deletionTaskStateKnown(task.DesiredState) {
		return fmt.Errorf("unknown task state for %q", taskID)
	}
	if cs := task.Spec.ContainerSpec; cs != nil && cs.StopGracePeriod != nil && *cs.StopGracePeriod < 0 {
		return fmt.Errorf("invalid negative grace for task %q", taskID)
	}
	if t.seen {
		if task.Version.Index == t.last.Version.Index && !reflect.DeepEqual(task, t.last) {
			return fmt.Errorf("conflicting observations at the same version for task %q", taskID)
		}
		newCID := deletionContainerID(task)
		if (t.nodeID != "" && task.NodeID != "" && t.nodeID != task.NodeID) || (t.containerID != "" && newCID != "" && t.containerID != newCID) || !reflect.DeepEqual(t.last.Spec, task.Spec) {
			return fmt.Errorf("changed assignment, container or immutable spec for task %q", taskID)
		}
		// REJECTED cannot coexist with observed execution, even when the
		// conflicting read is older. Version is not an internal SpecVersion.
		incomingStarted := task.Status.State == swarm.TaskStateStarting || task.Status.State == swarm.TaskStateRunning
		if (t.started && task.Status.State == swarm.TaskStateRejected) || (t.last.Status.State == swarm.TaskStateRejected && incomingStarted) {
			return fmt.Errorf("REJECTED contradicts STARTING/RUNNING for task %q", taskID)
		}
		if task.Version.Index < t.last.Version.Index {
			olderProof := task.Status.State == swarm.TaskStateComplete || task.Status.State == swarm.TaskStateShutdown || task.Status.State == swarm.TaskStateRejected
			if (olderProof && deletionTaskMayExecute(t.last.Status.State)) || (incomingStarted && deletionTaskPreStart(t.last.Status.State)) {
				return fmt.Errorf("contradictory out-of-order task history for %q", taskID)
			}
			return nil // coherent stale reads must not overwrite newer evidence
		}
	}
	if t.started {
		switch task.Status.State {
		case swarm.TaskStateNew, swarm.TaskStateAllocated, swarm.TaskStatePending, swarm.TaskStateAssigned,
			swarm.TaskStateAccepted, swarm.TaskStatePreparing, swarm.TaskStateReady, swarm.TaskStateRejected:
			return fmt.Errorf("pre-start state after STARTING/RUNNING for task %q", taskID)
		}
		if t.last.Status.State == swarm.TaskStateRunning && task.Status.State == swarm.TaskStateStarting {
			return fmt.Errorf("STARTING after RUNNING for task %q", taskID)
		}
	}
	if t.proof != "" && task.Status.State != t.last.Status.State {
		return fmt.Errorf("state changed after exit/discard evidence for task %q", taskID)
	}
	t.last, t.seen, t.err = task, true, nil
	if task.NodeID != "" {
		t.nodeID = task.NodeID
	}
	if id := deletionContainerID(task); id != "" {
		t.containerID = id
	}
	switch task.Status.State {
	case swarm.TaskStateStarting, swarm.TaskStateRunning:
		t.started = true
	case swarm.TaskStateComplete:
		t.proof = "exit observed (COMPLETE)"
	case swarm.TaskStateShutdown:
		// The public API does not distinguish worker Shutdown from scheduler
		// discard. Both native routes support exit OR discard, not clean shutdown.
		t.proof = "exit or discard observed (SHUTDOWN)"
	case swarm.TaskStateRejected:
		t.proof = "pre-start discard observed (REJECTED)"
	}
	return nil
}

func deletionTaskPreStart(state swarm.TaskState) bool {
	switch state {
	case swarm.TaskStateNew, swarm.TaskStateAllocated, swarm.TaskStatePending, swarm.TaskStateAssigned,
		swarm.TaskStateAccepted, swarm.TaskStatePreparing, swarm.TaskStateReady:
		return true
	}
	return false
}

func deletionTaskMayExecute(state swarm.TaskState) bool {
	return deletionTaskPreStart(state) || state == swarm.TaskStateStarting || state == swarm.TaskStateRunning
}

func deletionTaskStateKnown(state swarm.TaskState) bool {
	switch state {
	case swarm.TaskStateNew, swarm.TaskStateAllocated, swarm.TaskStatePending, swarm.TaskStateAssigned,
		swarm.TaskStateAccepted, swarm.TaskStatePreparing, swarm.TaskStateReady, swarm.TaskStateStarting,
		swarm.TaskStateRunning, swarm.TaskStateComplete, swarm.TaskStateShutdown, swarm.TaskStateFailed,
		swarm.TaskStateRejected, swarm.TaskStateOrphaned, swarm.TaskStateRemove:
		return true
	}
	return false
}

func deletionContainerID(t swarm.Task) string {
	if t.Status.ContainerStatus != nil {
		return t.Status.ContainerStatus.ContainerID
	}
	return ""
}

// The worker paths inspected for Docker 28/29 discard protobuf Nanos. A nil
// grace delegates to worker/container policy which is not exposed by TaskInspect.
func serviceDeletionWindow(tasks []*serviceDeletionTask) (time.Duration, bool) {
	var maximum time.Duration
	for _, t := range tasks {
		if t.proof != "" {
			continue
		}
		cs := t.last.Spec.ContainerSpec
		if cs == nil || cs.StopGracePeriod == nil {
			return 0, false
		}
		grace := (*cs.StopGracePeriod / time.Second) * time.Second
		if grace > maximum {
			maximum = grace
		}
	}
	// R empty deliberately leaves only the observation margin for the object.
	const margin = 30 * time.Second
	const maxDuration = time.Duration(1<<63 - 1)
	if maximum > maxDuration-margin {
		return maxDuration, true
	}
	return maximum + margin, true
}

// Docker SDK v28 marks *all* HTTP 404s NotFound and does not expose response
// headers here. Require its typed category AND the exact native resource error;
// a generic route/HTML 404 is not evidence that the object is absent. This text
// discriminates metadata absence only, NEVER task exit/discard.
func serviceDeletionNotFound(err error, kind, id string) bool {
	if !errdefs.IsNotFound(err) {
		return false
	}
	for errors.Unwrap(err) != nil {
		err = errors.Unwrap(err)
	}
	return err.Error() == fmt.Sprintf("%s %s not found", kind, id)
}

func serviceDeletionRetryable(err error) bool {
	return client.IsErrConnectionFailed(err) || errdefs.IsInternal(err) || errdefs.IsUnavailable(err) || errdefs.IsDeadlineExceeded(err)
}

type serviceDeletionClock struct {
	now  func() time.Time
	wait func(context.Context, time.Duration) error
}

func newServiceDeletionClock() serviceDeletionClock {
	return serviceDeletionClock{now: time.Now, wait: func(ctx context.Context, d time.Duration) error {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return ctx.Err()
		}
	}}
}

type serviceDeletionAttempt struct {
	serviceID string
	phase     string
	object    string
	captured  bool
	tasks     []*serviceDeletionTask
	sdkOnly   bool
	deadline  time.Time
	clock     serviceDeletionClock
	nextTask  time.Time
}

func (a *serviceDeletionAttempt) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.deadline.IsZero() && !a.clock.now().Before(a.deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func (a *serviceDeletionAttempt) pause(ctx context.Context, delay time.Duration) error {
	if err := a.check(ctx); err != nil {
		return err
	}
	if remaining := a.deadline.Sub(a.clock.now()); !a.deadline.IsZero() && delay > remaining {
		delay = remaining
	}
	if err := a.clock.wait(ctx, delay); err != nil {
		return err
	}
	return a.check(ctx)
}

func (a *serviceDeletionAttempt) failure(err error) error {
	var pending []string
	for _, t := range a.tasks {
		if t.proof == "" {
			description := fmt.Sprintf("%s(state=%s, version=%d)", t.last.ID, t.last.Status.State, t.last.Version.Index)
			if t.err != nil {
				description += fmt.Sprintf(": %v", t.err)
			}
			pending = append(pending, description)
		}
	}
	budget := ""
	if a.sdkOnly {
		budget = " Grace could not be reconstructed for at least one pending task; only the received SDK context deadline was used (no additional timeout)."
	}
	return fmt.Errorf("service %s deletion verification unknown: phase=%s, object=%s, capture_complete=%t, captured=%d, satisfied=%d, pending=[%s]: %w.%s Exit/discard is not confirmed for unresolved tasks; they are not necessarily still running. Task evidence is invocation-local; refresh/restart can lose these checks", a.serviceID, a.phase, a.object, a.captured, len(a.tasks), len(a.tasks)-len(pending), strings.Join(pending, "; "), err, budget)
}

func (a *serviceDeletionAttempt) inspectService(ctx context.Context, cli *client.Client) error {
	if err := a.check(ctx); err != nil {
		return err
	}
	s, _, err := cli.ServiceInspectWithRaw(ctx, a.serviceID, swarm.ServiceInspectOptions{})
	if serviceDeletionNotFound(err, "service", a.serviceID) {
		a.object = "removed"
		return nil
	}
	if err != nil {
		a.object = "unknown"
		return err
	}
	if s.ID != a.serviceID {
		return fmt.Errorf("service identity mismatch: requested %q, received %q", a.serviceID, s.ID)
	}
	a.object = "present"
	return nil
}

func (a *serviceDeletionAttempt) inspectTask(ctx context.Context, cli *client.Client, t *serviceDeletionTask) error {
	// One in-flight read, >=20ms between task reads: bounded load without
	// goroutines or per-request timeouts. Each round visits every pending ID.
	if delay := a.nextTask.Sub(a.clock.now()); delay > 0 {
		if err := a.pause(ctx, delay); err != nil {
			return err
		}
	}
	if err := a.check(ctx); err != nil {
		return err
	}
	a.nextTask = a.clock.now().Add(20 * time.Millisecond)
	task, _, err := cli.TaskInspectWithRaw(ctx, t.last.ID)
	t.err = err
	if err != nil {
		if serviceDeletionNotFound(err, "task", t.last.ID) {
			return nil // unknown, retained in T, not proof
		}
		return err
	}
	return t.observe(task, a.serviceID, t.last.ID)
}

func deletionBackoff(n int) time.Duration {
	if n > 3 {
		n = 3
	}
	return 250 * time.Millisecond * time.Duration(1<<n)
}

func deleteService(ctx context.Context, serviceID string, cli *client.Client) error {
	return deleteServiceWithClock(ctx, serviceID, cli, newServiceDeletionClock())
}

func deleteServiceWithClock(ctx context.Context, serviceID string, cli *client.Client, clock serviceDeletionClock) error {
	a := &serviceDeletionAttempt{serviceID: serviceID, phase: "capture", object: "unknown", clock: clock}
	deadline, bounded := ctx.Deadline()
	if !bounded || strings.TrimSpace(serviceID) == "" {
		return a.failure(fmt.Errorf("a nonempty original ServiceID and a bounded SDK context are required"))
	}
	a.deadline = deadline // capture/preobservation use only the received context
	retryRead := func(read func() error) error {
		for n := 0; ; n++ {
			if err := a.check(ctx); err != nil {
				return err
			}
			err := read()
			if err == nil || !serviceDeletionRetryable(err) {
				return err
			}
			if err := a.pause(ctx, deletionBackoff(n)); err != nil {
				return err
			}
		}
	}
	if err := retryRead(func() error { return a.inspectService(ctx, cli) }); err != nil {
		return a.failure(err)
	}
	if a.object == "removed" {
		return a.failure(fmt.Errorf("original service initially absent; no task capture from this invocation"))
	}
	var snapshot []swarm.Task
	if err := retryRead(func() (err error) {
		snapshot, err = cli.TaskList(ctx, swarm.TaskListOptions{Filters: filters.NewArgs(filters.Arg("service", serviceID))})
		return err
	}); err != nil {
		return a.failure(err)
	}
	if snapshot == nil {
		return a.failure(fmt.Errorf("invalid null task list; a complete capture is required"))
	}
	byID := make(map[string]*serviceDeletionTask, len(snapshot))
	for _, task := range snapshot {
		t, exists := byID[task.ID]
		if !exists {
			t = &serviceDeletionTask{}
			byID[task.ID] = t
			a.tasks = append(a.tasks, t)
		}
		if err := t.observe(task, serviceID, task.ID); err != nil {
			return a.failure(err)
		}
	}
	a.captured = true
	a.phase = "preobservation"
	queue := make([]*serviceDeletionTask, 0, len(a.tasks))
	for _, t := range a.tasks {
		if t.proof == "" {
			queue = append(queue, t)
		}
	}
	for round := 0; len(queue) > 0; round++ {
		var retryQueue []*serviceDeletionTask
		for _, t := range queue {
			if err := a.inspectTask(ctx, cli, t); err != nil {
				if !serviceDeletionRetryable(err) {
					return a.failure(err)
				}
				retryQueue = append(retryQueue, t)
			}
		}
		queue = retryQueue
		if len(queue) > 0 {
			if err := a.pause(ctx, deletionBackoff(round)); err != nil {
				return a.failure(err)
			}
		}
	}
	if err := a.check(ctx); err != nil {
		return a.failure(err)
	}
	// Fix the window once, immediately before the FIRST DELETE. All resends,
	// reconciliation, partial progress and observations share this same deadline.
	window, known := serviceDeletionWindow(a.tasks)
	a.sdkOnly = !known
	if a.sdkOnly {
		log.Printf("[INFO] Service %s: pending task grace could not be reconstructed; deletion observation uses only the received SDK context deadline", serviceID)
	}
	if known {
		if candidate := clock.now().Add(window); candidate.Before(a.deadline) {
			a.deadline = candidate
		}
	}
	phaseCtx, cancel := context.WithDeadline(ctx, a.deadline)
	defer cancel()
	a.phase = "delete/reconcile"
	for sends := 0; ; {
		if err := a.check(phaseCtx); err != nil {
			return a.failure(err)
		}
		err := cli.ServiceRemove(phaseCtx, serviceID)
		sends++
		if err == nil {
			a.object = "removed"
			break
		}
		// Even a failed DELETE may have taken effect. Never resend without a
		// positive inspect of the original identity, including after a 404.
		a.object = "unknown"
		deleteErr := err
		for n := 0; ; n++ {
			err = a.inspectService(phaseCtx, cli)
			if err == nil {
				break
			}
			if !serviceDeletionRetryable(err) {
				return a.failure(fmt.Errorf("DELETE: %v; reconciliation: %w", deleteErr, err))
			}
			if err := a.pause(phaseCtx, deletionBackoff(n)); err != nil {
				return a.failure(fmt.Errorf("DELETE: %v; reconciliation: %w", deleteErr, err))
			}
		}
		if a.object == "removed" {
			break
		}
		if (!serviceDeletionRetryable(deleteErr) && !serviceDeletionNotFound(deleteErr, "service", serviceID)) || sends >= 3 {
			return a.failure(fmt.Errorf("DELETE rejected or resend limit reached with original service present: %w", deleteErr))
		}
		if err := a.pause(phaseCtx, deletionBackoff(sends-1)); err != nil {
			return a.failure(err)
		}
	}
	a.phase = "task observation"
	for failures := 0; ; {
		pending, transient := false, false
		for _, t := range a.tasks {
			if t.proof != "" {
				continue
			}
			if err := a.inspectTask(phaseCtx, cli, t); err != nil {
				if !serviceDeletionRetryable(err) {
					return a.failure(err)
				}
				transient = true
			}
			if t.err != nil {
				transient = true // includes task 404: retry metadata, never infer exit
			}
			pending = pending || t.proof == ""
		}
		if err := a.check(phaseCtx); err != nil {
			return a.failure(err)
		}
		if !pending {
			log.Printf("[INFO] Service %s deleted; exit/discard evidence retained for %d captured tasks (invocation-local, not global closure)", serviceID, len(a.tasks))
			return nil
		}
		delay := 100 * time.Millisecond
		if transient {
			delay = deletionBackoff(failures)
			failures++
		} else {
			failures = 0
		}
		if err := a.pause(phaseCtx, delay); err != nil {
			return a.failure(err)
		}
	}
}
