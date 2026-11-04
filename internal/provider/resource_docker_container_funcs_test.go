package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestDockerContainer_DeleteMissingContainer verifies that deleting a
// container that no longer exists on the Docker daemon succeeds.
func TestDockerContainer_DeleteMissingContainer(t *testing.T) {
	meta := &ProviderConfig{
		DefaultConfig: &Config{
			Host: "unix:///var/run/docker.sock",
		},
	}

	raw := map[string]interface{}{
		"name":                  "nonexistent",
		"image":                 "sha256:deadbeef",
		"attach":                false,
		"destroy_grace_seconds": 0,
		"remove_volumes":        true,
		"rm":                    false,
	}
	d := schema.TestResourceDataRaw(t, resourceDockerContainer().Schema, raw)
	d.SetId("nonexistent_container_id")

	diags := resourceDockerContainerDelete(context.Background(), d, meta)
	if diags.HasError() {
		t.Fatalf("expected no error deleting missing container, got: %v", diags)
	}
}

// fakeContainerEngine is a minimal Docker API server for container create
// tests. Its create endpoint records the container under the requested name
// and then either answers, fails, or holds the response until the client
// goes away.
type fakeContainerEngine struct {
	mu         sync.Mutex
	containers map[string]string // name -> ID
	holdCreate bool
	failCreate bool
	created    chan struct{}
}

// serve starts the fake engine and returns a provider config pointing at it.
func (e *fakeContainerEngine) serve(t *testing.T) *ProviderConfig {
	t.Helper()
	const imageID = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if e.containers == nil {
		e.containers = map[string]string{}
	}
	e.created = make(chan struct{})

	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/_ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("API-Version", "1.47")
		w.Header().Set("OSType", "linux")
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /{version}/images/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]any{{"Id": imageID}})
	})
	mux.HandleFunc("GET /{version}/images/{name}/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"Id": imageID})
	})
	mux.HandleFunc("GET /{version}/containers/{name}/json", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		e.mu.Lock()
		id, ok := e.containers[name]
		e.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "No such container: " + name})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"Id": id, "Name": "/" + name})
	})
	mux.HandleFunc("POST /{version}/containers/create", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		// Consume the body so the server notices when the client disconnects.
		_, _ = io.Copy(io.Discard, r.Body)
		e.mu.Lock()
		if _, taken := e.containers[name]; !taken {
			e.containers[name] = "created-" + name
		}
		id := e.containers[name]
		e.mu.Unlock()
		switch {
		case e.holdCreate:
			close(e.created)
			<-r.Context().Done()
		case e.failCreate:
			writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "create failed after recording the container"})
		default:
			writeJSON(w, http.StatusCreated, map[string]any{"Id": id})
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &ProviderConfig{
		DefaultConfig: &Config{
			Host:                     "tcp://" + srv.Listener.Addr().String(),
			DisableDockerDaemonCheck: true,
		},
	}
}

// create runs resourceDockerContainerCreate against the fake engine. When the
// engine holds the create response, ctx is cancelled once the engine has
// recorded the container, like an interrupted apply.
func (e *fakeContainerEngine) create(t *testing.T, name string) (*schema.ResourceData, diag.Diagnostics) {
	t.Helper()
	meta := e.serve(t)
	d := schema.TestResourceDataRaw(t, resourceDockerContainer().Schema, map[string]any{
		"name":  name,
		"image": "busybox:latest",
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	if e.holdCreate {
		go func() {
			select {
			case <-e.created:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	return d, resourceDockerContainerCreate(ctx, d, meta)
}

// TestDockerContainer_CreateInterrupted verifies that a container the daemon
// created while the create request was interrupted is recorded in state.
func TestDockerContainer_CreateInterrupted(t *testing.T) {
	e := &fakeContainerEngine{holdCreate: true}

	d, diags := e.create(t, "interrupted")
	if !diags.HasError() {
		t.Fatal("expected an error from the interrupted create")
	}
	if got, want := d.Id(), "created-interrupted"; got != want {
		t.Fatalf("expected the created container to be recorded: got ID %q, want %q", got, want)
	}
}

// TestDockerContainer_CreateInterruptedNameTaken verifies that an interrupted
// create does not adopt a container that already had the name.
func TestDockerContainer_CreateInterruptedNameTaken(t *testing.T) {
	e := &fakeContainerEngine{
		holdCreate: true,
		containers: map[string]string{"taken": "someone-elses-container"},
	}

	d, diags := e.create(t, "taken")
	if !diags.HasError() {
		t.Fatal("expected an error from the interrupted create")
	}
	if got := d.Id(); got != "" {
		t.Fatalf("expected no container to be recorded, got ID %q", got)
	}
}

// TestDockerContainer_CreateFailed verifies that a create error without an
// interruption leaves the container unrecorded, as before.
func TestDockerContainer_CreateFailed(t *testing.T) {
	e := &fakeContainerEngine{failCreate: true}

	d, diags := e.create(t, "failed")
	if !diags.HasError() {
		t.Fatal("expected an error from the failed create")
	}
	if got := d.Id(); got != "" {
		t.Fatalf("expected no container to be recorded, got ID %q", got)
	}
}

func TestCopyContainerLogs_Demultiplex(t *testing.T) {
	var input bytes.Buffer
	stdoutWriter := stdcopy.NewStdWriter(&input, stdcopy.Stdout)
	stderrWriter := stdcopy.NewStdWriter(&input, stdcopy.Stderr)

	if _, err := stdoutWriter.Write([]byte("stdout\n")); err != nil {
		t.Fatalf("failed to write stdout log frame: %v", err)
	}
	if _, err := stderrWriter.Write([]byte("stderr\n")); err != nil {
		t.Fatalf("failed to write stderr log frame: %v", err)
	}

	var output bytes.Buffer
	if err := copyContainerLogs(&output, &input, false); err != nil {
		t.Fatalf("expected logs to be copied, got error: %v", err)
	}

	if got, want := output.String(), "stdout\nstderr\n"; got != want {
		t.Fatalf("unexpected logs output: got %q, want %q", got, want)
	}
}

func TestCopyContainerLogs_TTY(t *testing.T) {
	input := bytes.NewBufferString("tty-output\n")
	var output bytes.Buffer

	if err := copyContainerLogs(&output, input, true); err != nil {
		t.Fatalf("expected logs to be copied, got error: %v", err)
	}

	if got, want := output.String(), "tty-output\n"; got != want {
		t.Fatalf("unexpected logs output: got %q, want %q", got, want)
	}
}

func TestResolveUploadID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		field string
		want  int
	}{
		{name: "empty", field: "owner"},
		{name: "numeric", value: "1234", field: "owner", want: 1234},
		{name: "owner name", value: "root", field: "owner", want: 0},
		{name: "group name", value: "root", field: "group", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveUploadID(test.value, test.field)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected ID %d, got %d", test.want, got)
			}
		})
	}

	if _, err := resolveUploadID("nonexistent_user_xyz", "owner"); err == nil {
		t.Fatal("expected an error for an unknown owner")
	}
	if _, err := resolveUploadID("nonexistent_group_xyz", "group"); err == nil {
		t.Fatal("expected an error for an unknown group")
	}
}
