# Core Libraries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the two libraries — `internal/capi` (CAPI cluster discovery + workload kubeconfig fetch) and `internal/shell` (per-shell init-script generation + spawn helpers) — that the TUI (Plan 3) and Web backend (Plan 4) will both consume. No user-facing UX in this plan; outcome is well-tested Go packages with clean APIs.

**Architecture:**
- `internal/capi` provides a `Client` that wraps `client-go` + the CAPI v1beta1 scheme. Methods: `ListClusters`, `WatchClusters` (event channel), `FetchKubeconfig`. Uses `controller-runtime/pkg/client/fake` for unit tests; integration tests against a real kind cluster live behind a build tag and a Makefile target so they don't run in `make test` by default.
- `internal/shell` is a building-block library. `PrepareSpawn(opts) (*SpawnPlan, error)` returns a ready-to-Start `*exec.Cmd` plus a `Cleanup` func. Per-shell init logic (bash `--rcfile`, zsh `ZDOTDIR`, fish `--init-command`) lives in dedicated unexported helpers; each is independently testable. Caller (TUI or web) owns stdio wiring or PTY allocation.

**Tech Stack:**
- `sigs.k8s.io/cluster-api/api/v1beta1` — CAPI types
- `k8s.io/client-go` — REST config, dynamic client, informers
- `k8s.io/apimachinery` — runtime scheme, watch types
- `sigs.k8s.io/controller-runtime/pkg/client/fake` — test fake
- `k8s.io/api/core/v1` — `Secret` type for kubeconfig fetch
- existing: `gopkg.in/yaml.v3`, `github.com/stretchr/testify`

**End-state of this plan:**
```go
// All compile and unit tests pass; integration test target documented but not on default path.
import (
    "github.com/v4run/kapish/internal/capi"
    "github.com/v4run/kapish/internal/shell"
)
```

---

## Task 1: Add CAPI + client-go dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add dependencies**

Run:
```sh
go get sigs.k8s.io/cluster-api/api/v1beta1@latest
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest
go get k8s.io/api@latest
go get sigs.k8s.io/controller-runtime@latest
```

Expected: each prints `go: added <module> v<...>`.

- [ ] **Step 2: Tidy and verify build**

Run: `go mod tidy && go build ./...`
Expected: success, no errors.

- [ ] **Step 3: Run existing tests**

Run: `go test ./... -count=1`
Expected: all packages still green (none reach the new deps yet).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add CAPI, client-go, controller-runtime dependencies"
```

---

## Task 2: Define `internal/capi.Cluster` type and conversion from v1beta1

**Files:**
- Create: `internal/capi/types.go`
- Create: `internal/capi/types_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/capi/types_test.go`:

```go
package capi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestFromV1Beta1_PopulatesFields(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "prod-eu-1",
			Namespace:         "prod",
			CreationTimestamp: metav1.NewTime(created),
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				Kind: "AWSCluster",
			},
			Topology: &clusterv1.Topology{Version: "v1.30.2"},
		},
		Status: clusterv1.ClusterStatus{
			Phase:               "Provisioned",
			ControlPlaneReady:   true,
			InfrastructureReady: true,
		},
	}

	c := FromV1Beta1(v1)
	assert.Equal(t, "prod-eu-1", c.Name)
	assert.Equal(t, "prod", c.Namespace)
	assert.Equal(t, "Provisioned", c.Phase)
	assert.True(t, c.ControlPlaneReady)
	assert.True(t, c.InfrastructureReady)
	assert.Equal(t, "v1.30.2", c.K8sVersion)
	assert.Equal(t, "aws", c.Provider)
	assert.Equal(t, created, c.CreationTimestamp)
}

func TestFromV1Beta1_EmptyInfrastructureRef(t *testing.T) {
	v1 := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "no-infra"},
		Status:     clusterv1.ClusterStatus{Phase: "Pending"},
	}
	c := FromV1Beta1(v1)
	assert.Equal(t, "", c.Provider, "empty InfrastructureRef -> empty provider")
	assert.Equal(t, "", c.K8sVersion, "no Topology -> empty K8sVersion")
}

func TestProviderFromKind(t *testing.T) {
	cases := map[string]string{
		"AWSCluster":     "aws",
		"GCPCluster":     "gcp",
		"AzureCluster":   "azure",
		"VSphereCluster": "vsphere",
		"HetznerCluster": "hetzner",
		"OpenStackCluster": "openstack",
		"":               "",
	}
	for kind, want := range cases {
		require.Equal(t, want, providerFromKind(kind), "kind=%s", kind)
	}
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/capi -v`
Expected: FAIL — package or `FromV1Beta1`/`providerFromKind` undefined.

- [ ] **Step 3: Implement types.go**

Create `internal/capi/types.go`:

```go
// Package capi wraps Cluster API types and exposes a small, focused API
// for kapish: list/watch CAPI Cluster CRDs and fetch workload kubeconfigs.
package capi

import (
	"strings"
	"time"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Cluster is kapish's view of a CAPI Cluster. We don't expose the full
// v1beta1 type to consumers — TUI / Web only need a stable subset.
type Cluster struct {
	Name      string
	Namespace string

	Phase string // Pending | Provisioning | Provisioned | Deleting | Failed | ""

	ControlPlaneReady   bool
	InfrastructureReady bool

	// K8sVersion is best-effort. spec.topology.version when ClusterClass is in
	// use; otherwise empty (the version lives on the referenced control-plane
	// object, which we'd need a second GET to resolve — deferred).
	K8sVersion string

	// Provider is derived from spec.infrastructureRef.kind:
	//   AWSCluster -> "aws", GCPCluster -> "gcp", AzureCluster -> "azure",
	//   VSphereCluster -> "vsphere", HetznerCluster -> "hetzner", etc.
	// "" if InfrastructureRef is missing or unrecognized.
	Provider string

	CreationTimestamp time.Time
}

// FromV1Beta1 converts a CAPI v1beta1.Cluster into kapish's Cluster.
func FromV1Beta1(v *clusterv1.Cluster) Cluster {
	c := Cluster{
		Name:                v.Name,
		Namespace:           v.Namespace,
		Phase:               v.Status.Phase,
		ControlPlaneReady:   v.Status.ControlPlaneReady,
		InfrastructureReady: v.Status.InfrastructureReady,
		CreationTimestamp:   v.CreationTimestamp.Time,
	}
	if v.Spec.Topology != nil {
		c.K8sVersion = v.Spec.Topology.Version
	}
	if v.Spec.InfrastructureRef != nil {
		c.Provider = providerFromKind(v.Spec.InfrastructureRef.Kind)
	}
	return c
}

// providerFromKind extracts a short provider tag from the InfrastructureRef
// kind. Unknown kinds return "".
func providerFromKind(kind string) string {
	if kind == "" {
		return ""
	}
	// Convention: <Provider>Cluster (AWSCluster, GCPCluster, AzureCluster,
	// VSphereCluster, HetznerCluster, OpenStackCluster, ...).
	const suffix = "Cluster"
	if !strings.HasSuffix(kind, suffix) {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(kind, suffix))
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/capi -v`
Expected: PASS for all 3 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/capi/types.go internal/capi/types_test.go
git commit -m "feat(capi): Cluster type and FromV1Beta1 conversion"
```

---

## Task 3: Build a `Client` with kubeconfig+context resolution

**Files:**
- Create: `internal/capi/client.go`
- Create: `internal/capi/client_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/capi/client_test.go`:

```go
package capi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tinyKubeconfig = `apiVersion: v1
kind: Config
current-context: test-ctx
clusters:
- name: c1
  cluster:
    server: https://localhost:6443
contexts:
- name: test-ctx
  context:
    cluster: c1
    user: test-user
- name: alt-ctx
  context:
    cluster: c1
    user: test-user
users:
- name: test-user
  user:
    token: redacted
`

func TestNewClient_LoadsKubeconfigCurrentContext(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(tinyKubeconfig), 0o600))

	c, err := NewClient(Options{
		Kubeconfig: kubeconfig,
	})
	require.NoError(t, err)
	assert.Equal(t, "test-ctx", c.Context())
}

func TestNewClient_OverrideContext(t *testing.T) {
	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfig, []byte(tinyKubeconfig), 0o600))

	c, err := NewClient(Options{
		Kubeconfig: kubeconfig,
		Context:    "alt-ctx",
	})
	require.NoError(t, err)
	assert.Equal(t, "alt-ctx", c.Context())
}

func TestNewClient_MissingKubeconfig(t *testing.T) {
	_, err := NewClient(Options{Kubeconfig: "/totally/nope/kubeconfig"})
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/capi -run TestNewClient -v`
Expected: FAIL — `NewClient` and `Options` undefined.

- [ ] **Step 3: Implement client.go**

Create `internal/capi/client.go`:

```go
package capi

import (
	"errors"
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Options configure NewClient.
type Options struct {
	// Kubeconfig is the path to the kubeconfig file. Empty falls back to
	// $KUBECONFIG / ~/.kube/config (clientcmd default loading rules).
	Kubeconfig string

	// Context overrides kubeconfig's current-context. Empty = current-context.
	Context string

	// Namespace narrows list/watch to a single namespace. Empty = all namespaces.
	Namespace string
}

// Client is kapish's wrapper over client-go for talking to a CAPI mgmt cluster.
// It holds a dynamic client (for CAPI Cluster CRDs, which we access by GVR
// rather than typed clients) and a typed kubernetes client (for Secret reads).
type Client struct {
	dyn       dynamic.Interface
	kube      kubernetes.Interface
	rest      *rest.Config
	context   string
	namespace string
}

// NewClient resolves kubeconfig+context, builds clients, and returns the wrapper.
func NewClient(opts Options) (*Client, error) {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		loader.ExplicitPath = opts.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)

	rawCfg, err := cfg.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("capi: load kubeconfig: %w", err)
	}
	ctxName := opts.Context
	if ctxName == "" {
		ctxName = rawCfg.CurrentContext
	}
	if ctxName == "" {
		return nil, errors.New("capi: kubeconfig has no current-context and none specified")
	}
	if _, ok := rawCfg.Contexts[ctxName]; !ok {
		return nil, fmt.Errorf("capi: context %q not found in kubeconfig", ctxName)
	}

	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("capi: build REST config: %w", err)
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("capi: build dynamic client: %w", err)
	}
	kube, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("capi: build kubernetes client: %w", err)
	}

	return &Client{
		dyn:       dyn,
		kube:      kube,
		rest:      restCfg,
		context:   ctxName,
		namespace: opts.Namespace,
	}, nil
}

// Context reports the resolved kubeconfig context this Client uses.
func (c *Client) Context() string { return c.context }

// Namespace reports the namespace scope ("" = all).
func (c *Client) Namespace() string { return c.namespace }
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/capi -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/capi/client.go internal/capi/client_test.go
git commit -m "feat(capi): Client wrapping client-go + clientcmd"
```

---

## Task 4: List CAPI Clusters via dynamic client

**Files:**
- Modify: `internal/capi/client.go`
- Create: `internal/capi/list.go`
- Create: `internal/capi/list_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/capi/list_test.go`:

```go
package capi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestListClusters_ReturnsAllNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clusterv1.AddToScheme(scheme))

	a := newFakeCluster("a", "ns1", "Provisioned")
	b := newFakeCluster("b", "ns2", "Pending")
	dyn := fake.NewSimpleDynamicClient(scheme, a, b)

	c := &Client{dyn: dyn, namespace: ""}

	got, err := c.ListClusters(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)

	byName := map[string]Cluster{}
	for _, x := range got {
		byName[x.Name] = x
	}
	assert.Equal(t, "ns1", byName["a"].Namespace)
	assert.Equal(t, "Provisioned", byName["a"].Phase)
	assert.Equal(t, "ns2", byName["b"].Namespace)
	assert.Equal(t, "Pending", byName["b"].Phase)
}

func TestListClusters_NamespaceScoped(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clusterv1.AddToScheme(scheme))

	a := newFakeCluster("a", "ns1", "Provisioned")
	b := newFakeCluster("b", "ns2", "Pending")
	dyn := fake.NewSimpleDynamicClient(scheme, a, b)

	c := &Client{dyn: dyn, namespace: "ns1"}

	got, err := c.ListClusters(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "a", got[0].Name)
}

// helper: build a *clusterv1.Cluster with minimal status fields set.
func newFakeCluster(name, namespace, phase string) *clusterv1.Cluster {
	return &clusterv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: clusterv1.GroupVersion.String(),
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(time.Now().UTC().Truncate(time.Second)),
		},
		Status: clusterv1.ClusterStatus{Phase: phase},
	}
}

// gvr is the GVR for cluster.x-k8s.io/v1beta1 Cluster, exported for the test
// package to use. (The list.go file declares it.)
var _ = schema.GroupVersionResource{Group: "cluster.x-k8s.io", Version: "v1beta1", Resource: "clusters"}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/capi -run TestListClusters -v`
Expected: FAIL — `ListClusters` undefined.

- [ ] **Step 3: Implement list.go**

Create `internal/capi/list.go`:

```go
package capi

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// clusterGVR is the GroupVersionResource for cluster.x-k8s.io/v1beta1 Clusters.
// We use the dynamic client (rather than CAPI's typed client) so consumers
// don't transitively pull in the full CAPI client codegen.
var clusterGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta1",
	Resource: "clusters",
}

// ListClusters returns all CAPI Clusters in the configured namespace
// (or all namespaces if the Client's namespace is "").
func (c *Client) ListClusters(ctx context.Context) ([]Cluster, error) {
	li := c.dyn.Resource(clusterGVR)
	var (
		raw *runtime.Object
		err error
	)
	if c.namespace == "" {
		l, e := li.List(ctx, metav1.ListOptions{})
		if e != nil {
			return nil, fmt.Errorf("capi: list clusters: %w", e)
		}
		out := make([]Cluster, 0, len(l.Items))
		for i := range l.Items {
			cl := &clusterv1.Cluster{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(l.Items[i].UnstructuredContent(), cl); err != nil {
				return nil, fmt.Errorf("capi: convert cluster %s/%s: %w", l.Items[i].GetNamespace(), l.Items[i].GetName(), err)
			}
			out = append(out, FromV1Beta1(cl))
		}
		return out, nil
	}
	l, e := li.Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if e != nil {
		return nil, fmt.Errorf("capi: list clusters in %s: %w", c.namespace, e)
	}
	_ = raw
	_ = err
	out := make([]Cluster, 0, len(l.Items))
	for i := range l.Items {
		cl := &clusterv1.Cluster{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(l.Items[i].UnstructuredContent(), cl); err != nil {
			return nil, fmt.Errorf("capi: convert cluster %s/%s: %w", l.Items[i].GetNamespace(), l.Items[i].GetName(), err)
		}
		out = append(out, FromV1Beta1(cl))
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/capi -v`
Expected: PASS for all (existing + 2 new list tests).

- [ ] **Step 5: Commit**

```bash
git add internal/capi/list.go internal/capi/list_test.go
git commit -m "feat(capi): ListClusters via dynamic client"
```

---

## Task 5: Watch CAPI Cluster events

**Files:**
- Create: `internal/capi/watch.go`
- Create: `internal/capi/watch_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/capi/watch_test.go`:

```go
package capi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestWatchClusters_ReceivesAddedEvent(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clusterv1.AddToScheme(scheme))

	dyn := fake.NewSimpleDynamicClient(scheme)
	c := &Client{dyn: dyn, namespace: ""}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := c.WatchClusters(ctx)
	require.NoError(t, err)

	// Add a cluster via the fake client.
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, _ = dyn.Resource(clusterGVR).Namespace("default").Create(
			context.Background(),
			toUnstructured(t, newFakeCluster("new", "default", "Pending")),
			metav1ObjCreate(),
		)
	}()

	select {
	case ev := <-events:
		assert.Equal(t, EventAdded, ev.Type)
		assert.Equal(t, "new", ev.Cluster.Name)
		assert.Equal(t, "default", ev.Cluster.Namespace)
	case <-ctx.Done():
		t.Fatalf("did not receive Added event before timeout")
	}
}
```

Helper for the test (place at the bottom of `watch_test.go`):

```go
import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func toUnstructured(t *testing.T, obj any) *unstructured.Unstructured {
	t.Helper()
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: u}
}

func metav1ObjCreate() metav1.CreateOptions { return metav1.CreateOptions{} }
```

> Merge any duplicate imports with existing ones at the top of `watch_test.go`.

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/capi -run TestWatchClusters -v`
Expected: FAIL — `WatchClusters`, `Event`, `EventAdded` undefined.

- [ ] **Step 3: Implement watch.go**

Create `internal/capi/watch.go`:

```go
package capi

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// EventType classifies a cluster watch event.
type EventType int

const (
	EventAdded EventType = iota
	EventModified
	EventDeleted
	EventError
)

// Event is one cluster lifecycle event.
type Event struct {
	Type    EventType
	Cluster Cluster
	Err     error // populated only when Type == EventError
}

// WatchClusters returns a channel of Cluster events. The channel is closed
// when ctx is canceled, the watch errors out, or the underlying watcher
// is otherwise shut down. Caller is responsible for context lifecycle.
//
// On reconnect-worthy errors, kapish's higher layers (TUI/Web) should
// re-call WatchClusters with backoff.
func (c *Client) WatchClusters(ctx context.Context) (<-chan Event, error) {
	li := c.dyn.Resource(clusterGVR)
	var w watch.Interface
	var err error
	if c.namespace == "" {
		w, err = li.Watch(ctx, metav1.ListOptions{})
	} else {
		w, err = li.Namespace(c.namespace).Watch(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("capi: watch clusters: %w", err)
	}

	out := make(chan Event, 16)
	go func() {
		defer close(out)
		defer w.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.ResultChan():
				if !ok {
					return
				}
				kev := translateWatchEvent(ev)
				select {
				case <-ctx.Done():
					return
				case out <- kev:
				}
			}
		}
	}()
	return out, nil
}

func translateWatchEvent(ev watch.Event) Event {
	switch ev.Type {
	case watch.Added:
		return Event{Type: EventAdded, Cluster: clusterFromObject(ev.Object)}
	case watch.Modified:
		return Event{Type: EventModified, Cluster: clusterFromObject(ev.Object)}
	case watch.Deleted:
		return Event{Type: EventDeleted, Cluster: clusterFromObject(ev.Object)}
	case watch.Error:
		return Event{Type: EventError, Err: fmt.Errorf("capi watch error: %v", ev.Object)}
	}
	return Event{Type: EventError, Err: fmt.Errorf("capi: unknown watch event type %q", ev.Type)}
}

func clusterFromObject(obj runtime.Object) Cluster {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return Cluster{}
	}
	cl := &clusterv1.Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), cl); err != nil {
		return Cluster{}
	}
	return FromV1Beta1(cl)
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/capi -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/capi/watch.go internal/capi/watch_test.go
git commit -m "feat(capi): WatchClusters event stream"
```

---

## Task 6: Fetch workload kubeconfig from Secret

**Files:**
- Create: `internal/capi/kubeconfig.go`
- Create: `internal/capi/kubeconfig_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/capi/kubeconfig_test.go`:

```go
package capi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestFetchKubeconfig_HappyPath(t *testing.T) {
	const want = `apiVersion: v1
kind: Config
contexts: []
`
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-eu-1-kubeconfig",
			Namespace: "prod",
		},
		Data: map[string][]byte{"value": []byte(want)},
	}
	kube := fake.NewSimpleClientset(secret)
	c := &Client{kube: kube}

	got, err := c.FetchKubeconfig(context.Background(), "prod", "prod-eu-1")
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestFetchKubeconfig_MissingSecret(t *testing.T) {
	kube := fake.NewSimpleClientset()
	c := &Client{kube: kube}

	_, err := c.FetchKubeconfig(context.Background(), "prod", "missing")
	require.Error(t, err)
}

func TestFetchKubeconfig_MissingValueKey(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "weird-kubeconfig",
			Namespace: "prod",
		},
		Data: map[string][]byte{"otherKey": []byte("hi")},
	}
	kube := fake.NewSimpleClientset(secret)
	c := &Client{kube: kube}

	_, err := c.FetchKubeconfig(context.Background(), "prod", "weird")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/capi -run TestFetchKubeconfig -v`
Expected: FAIL — `FetchKubeconfig` undefined.

- [ ] **Step 3: Implement kubeconfig.go**

Create `internal/capi/kubeconfig.go`:

```go
package capi

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FetchKubeconfig reads the workload cluster's kubeconfig from the Secret
// named "<name>-kubeconfig" in the given namespace, with key "value".
// This is the CAPI convention.
func (c *Client) FetchKubeconfig(ctx context.Context, namespace, clusterName string) ([]byte, error) {
	secretName := clusterName + "-kubeconfig"
	s, err := c.kube.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("capi: get secret %s/%s: %w", namespace, secretName, err)
	}
	v, ok := s.Data["value"]
	if !ok {
		return nil, fmt.Errorf("capi: secret %s/%s has no 'value' key", namespace, secretName)
	}
	return v, nil
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/capi -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/capi/kubeconfig.go internal/capi/kubeconfig_test.go
git commit -m "feat(capi): FetchKubeconfig from <name>-kubeconfig Secret"
```

---

## Task 7: Shell `Options` and prompt-token rendering

**Files:**
- Create: `internal/shell/options.go`
- Create: `internal/shell/prompt.go`
- Create: `internal/shell/prompt_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/prompt_test.go`:

```go
package shell

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderPrompt_AllTokens(t *testing.T) {
	tok := PromptTokens{
		Cluster:  "prod-eu-1",
		Namespace: "prod",
		Provider: "aws",
		Ctx:      "mgmt-eu",
		Now:      time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC),
	}
	got := RenderPrompt("[{cluster}/{ns}] {provider}@{ctx} {time} ", tok)
	assert.Equal(t, "[prod-eu-1/prod] aws@mgmt-eu 14:30 ", got)
}

func TestRenderPrompt_EmptyTemplate(t *testing.T) {
	got := RenderPrompt("", PromptTokens{Cluster: "x"})
	assert.Equal(t, "", got)
}

func TestRenderPrompt_UnknownTokenLeftLiteral(t *testing.T) {
	got := RenderPrompt("[{nope}] ", PromptTokens{Cluster: "x"})
	assert.True(t, strings.Contains(got, "{nope}"), "got: %q", got)
}

func TestRenderPrompt_TimeIsHHMMLocal(t *testing.T) {
	got := RenderPrompt("{time}", PromptTokens{Now: time.Date(2026, 1, 1, 9, 5, 0, 0, time.Local)})
	assert.Equal(t, "09:05", got)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -v`
Expected: FAIL — package or `RenderPrompt`/`PromptTokens` undefined.

- [ ] **Step 3: Implement options.go**

Create `internal/shell/options.go`:

```go
// Package shell builds per-session shell init (rcfile / ZDOTDIR / fish init)
// and exposes a SpawnPlan that callers wrap in os/exec or PTY as appropriate.
package shell

import "time"

// Options describe everything needed to spawn a kapish shell session.
//
// PathToShell is the absolute path to the binary (e.g. "/bin/zsh"); empty
// means "use $SHELL". Cwd is optional. Env/Aliases/Prompt come from the
// merged kapish config. Kubeconfig is the bytes already fetched from the
// CAPI Secret (we write it to a temp file as part of PrepareSpawn).
//
// Promptokens carries the substitution values for the prompt template;
// callers populate it from the Cluster + management context.
type Options struct {
	PathToShell    string
	Cwd            string
	Env            map[string]string
	Aliases        map[string]string
	PromptTemplate string
	Kubeconfig     []byte
	PromptTokens   PromptTokens
}

// PromptTokens are the substitution values for the prompt template.
type PromptTokens struct {
	Cluster   string
	Namespace string
	Provider  string
	Ctx       string
	Now       time.Time // used to render {time}
}
```

- [ ] **Step 4: Implement prompt.go**

Create `internal/shell/prompt.go`:

```go
package shell

import "strings"

// RenderPrompt substitutes {cluster}, {ns}, {provider}, {ctx}, {time} in the
// template with values from tok. {time} renders as HH:MM local. Unknown
// tokens are left literal (validation happens at config-load time).
func RenderPrompt(tmpl string, tok PromptTokens) string {
	if tmpl == "" {
		return ""
	}
	r := strings.NewReplacer(
		"{cluster}", tok.Cluster,
		"{ns}", tok.Namespace,
		"{provider}", tok.Provider,
		"{ctx}", tok.Ctx,
		"{time}", tok.Now.Format("15:04"),
	)
	return r.Replace(tmpl)
}
```

- [ ] **Step 5: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS for all 4 prompt tests.

- [ ] **Step 6: Commit**

```bash
git add internal/shell/options.go internal/shell/prompt.go internal/shell/prompt_test.go
git commit -m "feat(shell): Options, PromptTokens, RenderPrompt"
```

---

## Task 8: Detect shell from $SHELL or Options

**Files:**
- Create: `internal/shell/detect.go`
- Create: `internal/shell/detect_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/detect_test.go`:

```go
package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_OptionsTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "zsh")
	require.NoError(t, os.WriteFile(fakeShell, []byte("#!/bin/sh\n"), 0o755))

	d, err := Detect(fakeShell)
	require.NoError(t, err)
	assert.Equal(t, fakeShell, d.Path)
	assert.Equal(t, KindZsh, d.Kind)
}

func TestDetect_FallsBackToShellEnv(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "bash")
	require.NoError(t, os.WriteFile(fakeShell, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("SHELL", fakeShell)

	d, err := Detect("")
	require.NoError(t, err)
	assert.Equal(t, fakeShell, d.Path)
	assert.Equal(t, KindBash, d.Kind)
}

func TestDetect_UnknownBasename(t *testing.T) {
	dir := t.TempDir()
	fakeShell := filepath.Join(dir, "ksh")
	require.NoError(t, os.WriteFile(fakeShell, []byte("#!/bin/sh\n"), 0o755))

	_, err := Detect(fakeShell)
	require.Error(t, err)
}

func TestDetect_NotInPath(t *testing.T) {
	_, err := Detect("/totally/not/here/zsh")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -run TestDetect -v`
Expected: FAIL — `Detect`, `Detected`, `KindBash`, `KindZsh`, `KindFish` undefined.

- [ ] **Step 3: Implement detect.go**

Create `internal/shell/detect.go`:

```go
package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Kind is the supported v1 shell flavor.
type Kind int

const (
	KindUnknown Kind = iota
	KindBash
	KindZsh
	KindFish
)

func (k Kind) String() string {
	switch k {
	case KindBash:
		return "bash"
	case KindZsh:
		return "zsh"
	case KindFish:
		return "fish"
	}
	return "unknown"
}

// Detected is the result of resolving a shell.
type Detected struct {
	Path string // absolute path to the shell binary
	Kind Kind
}

// Detect resolves the shell to use. Order of preference:
//   1. provided is non-empty (from kapish config or override)
//   2. $SHELL env var
// Returns an error if the resolved binary doesn't exist or has an
// unsupported basename.
func Detect(provided string) (Detected, error) {
	path := provided
	if path == "" {
		path = os.Getenv("SHELL")
	}
	if path == "" {
		return Detected{}, errors.New("shell: no shell provided and $SHELL is unset")
	}
	if _, err := os.Stat(path); err != nil {
		return Detected{}, fmt.Errorf("shell: %s not found: %w", path, err)
	}
	base := filepath.Base(path)
	var k Kind
	switch base {
	case "bash":
		k = KindBash
	case "zsh":
		k = KindZsh
	case "fish":
		k = KindFish
	default:
		return Detected{}, fmt.Errorf("shell: unsupported shell %q (v1: bash, zsh, fish)", base)
	}
	return Detected{Path: path, Kind: k}, nil
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/detect.go internal/shell/detect_test.go
git commit -m "feat(shell): Detect resolves shell from option or $SHELL"
```

---

## Task 9: bash init script generation

**Files:**
- Create: `internal/shell/bash.go`
- Create: `internal/shell/bash_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/bash_test.go`:

```go
package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashInit_HasStandardLines(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"FOO": "bar"},
		Aliases:        map[string]string{"k": "kubectl"},
		PromptTemplate: "[{cluster}] ",
		Cwd:            "/tmp",
		PromptTokens:   PromptTokens{Cluster: "x"},
	}
	got := bashInit(opts, "/tmp/kapish-abc/kubeconfig")

	assert.Contains(t, got, `[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"`)
	assert.Contains(t, got, `export KUBECONFIG="/tmp/kapish-abc/kubeconfig"`)
	assert.Contains(t, got, `export FOO='bar'`)
	assert.Contains(t, got, `alias k='kubectl'`)
	assert.Contains(t, got, `PS1='[x] '"$PS1"`)
	assert.Contains(t, got, `cd '/tmp'`)
}

func TestBashInit_EscapesSingleQuotes(t *testing.T) {
	opts := Options{
		Env:     map[string]string{"X": "it's tricky"},
		Aliases: map[string]string{"a": "echo 'hi'"},
		PromptTemplate: "",
	}
	got := bashInit(opts, "/k")

	// Envs and aliases should be safely single-quoted.
	assert.True(t, strings.Contains(got, `export X='it'\''s tricky'`),
		"expected ANSI-C-safe single-quote escaping, got:\n%s", got)
	assert.True(t, strings.Contains(got, `alias a='echo '\''hi'\'''`),
		"expected alias to escape inner single quotes, got:\n%s", got)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -run TestBashInit -v`
Expected: FAIL — `bashInit` undefined.

- [ ] **Step 3: Implement bash.go**

Create `internal/shell/bash.go`:

```go
package shell

import (
	"sort"
	"strings"
)

func bashInit(opts Options, kubeconfigPath string) string {
	var b strings.Builder

	// Source the user's normal rc first so existing aliases/functions stand.
	b.WriteString(`[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"` + "\n")

	// KUBECONFIG always set.
	b.WriteString(`export KUBECONFIG=` + posixSingleQuote(kubeconfigPath) + "\n")

	// Other env vars (sorted for stable output / tests).
	for _, k := range sortedKeys(opts.Env) {
		b.WriteString("export " + k + "=" + posixSingleQuote(opts.Env[k]) + "\n")
	}

	// Aliases (sorted).
	for _, k := range sortedKeys(opts.Aliases) {
		b.WriteString("alias " + k + "=" + posixSingleQuote(opts.Aliases[k]) + "\n")
	}

	// Prompt prefix (rendered before reaching here is fine, but to be safe
	// we render again from PromptTokens).
	if opts.PromptTemplate != "" {
		prefix := RenderPrompt(opts.PromptTemplate, opts.PromptTokens)
		b.WriteString("PS1=" + posixSingleQuote(prefix) + `"$PS1"` + "\n")
	}

	if opts.Cwd != "" {
		b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n")
	}

	return b.String()
}

// posixSingleQuote wraps s in single quotes, ANSI-C-escaping any embedded
// single quotes via the standard '\'' trick. Suitable for bash/zsh/fish
// rcfile generation; values are user-supplied so we must escape them.
func posixSingleQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/bash.go internal/shell/bash_test.go
git commit -m "feat(shell): bash init script generation"
```

---

## Task 10: zsh init script generation (ZDOTDIR strategy)

**Files:**
- Create: `internal/shell/zsh.go`
- Create: `internal/shell/zsh_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/zsh_test.go`:

```go
package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZshInit_HasStandardLines(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"FOO": "bar"},
		Aliases:        map[string]string{"k": "kubectl"},
		PromptTemplate: "[{cluster}] ",
		Cwd:            "",
		PromptTokens:   PromptTokens{Cluster: "x"},
	}
	got := zshInit(opts, "/tmp/kapish-abc/kubeconfig")

	assert.Contains(t, got, `[ -f "$HOME/.zshrc" ] && . "$HOME/.zshrc"`)
	assert.Contains(t, got, `export KUBECONFIG='/tmp/kapish-abc/kubeconfig'`)
	assert.Contains(t, got, `export FOO='bar'`)
	assert.Contains(t, got, `alias k='kubectl'`)
	assert.Contains(t, got, `PROMPT='[x] '"$PROMPT"`)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -run TestZshInit -v`
Expected: FAIL.

- [ ] **Step 3: Implement zsh.go**

Create `internal/shell/zsh.go`:

```go
package shell

import "strings"

func zshInit(opts Options, kubeconfigPath string) string {
	var b strings.Builder

	b.WriteString(`[ -f "$HOME/.zshrc" ] && . "$HOME/.zshrc"` + "\n")
	b.WriteString("export KUBECONFIG=" + posixSingleQuote(kubeconfigPath) + "\n")
	for _, k := range sortedKeys(opts.Env) {
		b.WriteString("export " + k + "=" + posixSingleQuote(opts.Env[k]) + "\n")
	}
	for _, k := range sortedKeys(opts.Aliases) {
		b.WriteString("alias " + k + "=" + posixSingleQuote(opts.Aliases[k]) + "\n")
	}
	if opts.PromptTemplate != "" {
		prefix := RenderPrompt(opts.PromptTemplate, opts.PromptTokens)
		b.WriteString("PROMPT=" + posixSingleQuote(prefix) + `"$PROMPT"` + "\n")
	}
	if opts.Cwd != "" {
		b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n")
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/zsh.go internal/shell/zsh_test.go
git commit -m "feat(shell): zsh init script generation"
```

---

## Task 11: fish init script generation

**Files:**
- Create: `internal/shell/fish.go`
- Create: `internal/shell/fish_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/fish_test.go`:

```go
package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFishInit_HasStandardLines(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"FOO": "bar"},
		Aliases:        map[string]string{"k": "kubectl"},
		PromptTemplate: "[{cluster}] ",
		Cwd:            "/tmp",
		PromptTokens:   PromptTokens{Cluster: "x"},
	}
	got := fishInit(opts, "/tmp/kapish-abc/kubeconfig")

	assert.Contains(t, got, "set -gx KUBECONFIG '/tmp/kapish-abc/kubeconfig'")
	assert.Contains(t, got, "set -gx FOO 'bar'")
	assert.Contains(t, got, "alias k 'kubectl'")
	// fish prompt is a function, not PS1.
	assert.Contains(t, got, "function fish_prompt")
	assert.Contains(t, got, "echo -n '[x] '")
	assert.Contains(t, got, "cd '/tmp'")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -run TestFishInit -v`
Expected: FAIL.

- [ ] **Step 3: Implement fish.go**

Create `internal/shell/fish.go`:

```go
package shell

import "strings"

func fishInit(opts Options, kubeconfigPath string) string {
	var b strings.Builder

	b.WriteString("set -gx KUBECONFIG " + posixSingleQuote(kubeconfigPath) + "\n")
	for _, k := range sortedKeys(opts.Env) {
		b.WriteString("set -gx " + k + " " + posixSingleQuote(opts.Env[k]) + "\n")
	}
	for _, k := range sortedKeys(opts.Aliases) {
		b.WriteString("alias " + k + " " + posixSingleQuote(opts.Aliases[k]) + "\n")
	}
	if opts.PromptTemplate != "" {
		prefix := RenderPrompt(opts.PromptTemplate, opts.PromptTokens)
		b.WriteString("functions -c fish_prompt _kapish_orig_prompt 2>/dev/null\n")
		b.WriteString("function fish_prompt\n")
		b.WriteString("    echo -n " + posixSingleQuote(prefix) + "\n")
		b.WriteString("    if functions -q _kapish_orig_prompt; _kapish_orig_prompt; end\n")
		b.WriteString("end\n")
	}
	if opts.Cwd != "" {
		b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n")
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/fish.go internal/shell/fish_test.go
git commit -m "feat(shell): fish init script generation"
```

---

## Task 12: Per-session temp dir + kubeconfig file lifecycle

**Files:**
- Create: `internal/shell/tempdir.go`
- Create: `internal/shell/tempdir_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/tempdir_test.go`:

```go
package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionDir_CreatesAndWritesKubeconfig(t *testing.T) {
	d, err := newSessionDir([]byte("# kubeconfig content\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Remove() })

	// Directory exists with 0700.
	fi, err := os.Stat(d.Path)
	require.NoError(t, err)
	assert.True(t, fi.IsDir())
	assert.Equal(t, os.FileMode(0o700), fi.Mode().Perm())

	// kubeconfig present with 0600 and the content.
	got, err := os.ReadFile(d.KubeconfigPath)
	require.NoError(t, err)
	assert.Equal(t, "# kubeconfig content\n", string(got))

	kfi, err := os.Stat(d.KubeconfigPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), kfi.Mode().Perm())

	// Path lives under os.TempDir() with the kapish- prefix.
	rel, err := filepath.Rel(os.TempDir(), d.Path)
	require.NoError(t, err)
	assert.False(t, filepath.IsAbs(rel))
	assert.Contains(t, filepath.Base(d.Path), "kapish-")
}

func TestSessionDir_RemoveIsIdempotent(t *testing.T) {
	d, err := newSessionDir(nil)
	require.NoError(t, err)
	require.NoError(t, d.Remove())
	require.NoError(t, d.Remove(), "second Remove must not error")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -run "TestNewSessionDir_|TestSessionDir_" -v`
Expected: FAIL — `newSessionDir`, `SessionDir` undefined.

- [ ] **Step 3: Implement tempdir.go**

Create `internal/shell/tempdir.go`:

```go
package shell

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SessionDir is a per-spawn temp dir that holds the kubeconfig and any
// shell init files. Caller MUST call Remove() on shell exit (use defer).
type SessionDir struct {
	Path           string
	KubeconfigPath string
	removed        bool
}

func newSessionDir(kubeconfig []byte) (*SessionDir, error) {
	dir, err := os.MkdirTemp("", "kapish-*")
	if err != nil {
		return nil, fmt.Errorf("shell: mkdtemp: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("shell: chmod %s: %w", dir, err)
	}
	kpath := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(kpath, kubeconfig, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("shell: write kubeconfig: %w", err)
	}
	return &SessionDir{Path: dir, KubeconfigPath: kpath}, nil
}

// Remove deletes the temp dir. Idempotent: a second call returns nil.
func (d *SessionDir) Remove() error {
	if d.removed {
		return nil
	}
	d.removed = true
	if err := os.RemoveAll(d.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("shell: cleanup %s: %w", d.Path, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/tempdir.go internal/shell/tempdir_test.go
git commit -m "feat(shell): SessionDir for per-spawn temp dir + kubeconfig"
```

---

## Task 13: PrepareSpawn — assemble exec.Cmd for any shell

**Files:**
- Create: `internal/shell/spawn.go`
- Create: `internal/shell/spawn_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/shell/spawn_test.go`:

```go
package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareSpawn_BashUsesRcfileFlag(t *testing.T) {
	dir := t.TempDir()
	bash := filepath.Join(dir, "bash")
	require.NoError(t, os.WriteFile(bash, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{
		PathToShell: bash,
		Env:         map[string]string{"FOO": "bar"},
	}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	require.NotNil(t, plan.Cmd)
	assert.Equal(t, bash, plan.Cmd.Path)
	// Must include --rcfile <path> in args.
	args := plan.Cmd.Args
	require.True(t, len(args) >= 3, "expected --rcfile + path in args: %v", args)
	rcIdx := -1
	for i, a := range args {
		if a == "--rcfile" {
			rcIdx = i
		}
	}
	require.True(t, rcIdx >= 0 && rcIdx+1 < len(args), "expected --rcfile flag: %v", args)
	rcfile := args[rcIdx+1]
	body, err := os.ReadFile(rcfile)
	require.NoError(t, err)
	assert.Contains(t, string(body), "export FOO='bar'")
	assert.Contains(t, string(body), "export KUBECONFIG=")
}

func TestPrepareSpawn_ZshSetsZDOTDIR(t *testing.T) {
	dir := t.TempDir()
	zsh := filepath.Join(dir, "zsh")
	require.NoError(t, os.WriteFile(zsh, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{PathToShell: zsh}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	// ZDOTDIR must be set in plan.Cmd.Env to the session dir.
	var zdot string
	for _, e := range plan.Cmd.Env {
		if strings.HasPrefix(e, "ZDOTDIR=") {
			zdot = strings.TrimPrefix(e, "ZDOTDIR=")
		}
	}
	require.NotEmpty(t, zdot, "ZDOTDIR must be set in env")
	body, err := os.ReadFile(filepath.Join(zdot, ".zshrc"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "export KUBECONFIG=")
}

func TestPrepareSpawn_FishUsesInitCommand(t *testing.T) {
	dir := t.TempDir()
	fish := filepath.Join(dir, "fish")
	require.NoError(t, os.WriteFile(fish, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{PathToShell: fish}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	args := plan.Cmd.Args
	hasInit := false
	for _, a := range args {
		if strings.HasPrefix(a, "--init-command=") {
			hasInit = true
			assert.Contains(t, a, "set -gx KUBECONFIG")
		}
	}
	assert.True(t, hasInit, "fish must be invoked with --init-command=...")
}

func TestPrepareSpawn_CleanupRemovesDir(t *testing.T) {
	dir := t.TempDir()
	bash := filepath.Join(dir, "bash")
	require.NoError(t, os.WriteFile(bash, []byte("#!/bin/sh\n"), 0o755))

	plan, err := PrepareSpawn(Options{PathToShell: bash}, []byte("# kc\n"))
	require.NoError(t, err)

	sessionPath := plan.SessionDir.Path
	require.NoError(t, plan.Cleanup())
	_, err = os.Stat(sessionPath)
	assert.Error(t, err, "session dir must be gone after Cleanup")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./internal/shell -run TestPrepareSpawn -v`
Expected: FAIL — `PrepareSpawn`, `SpawnPlan` undefined.

- [ ] **Step 3: Implement spawn.go**

Create `internal/shell/spawn.go`:

```go
package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SpawnPlan is everything needed to launch a kapish shell. Caller wires
// stdio (or wraps in PTY) and calls Start/Run on Cmd. Cleanup MUST be called
// (defer it) to remove the temp dir.
type SpawnPlan struct {
	Cmd        *exec.Cmd
	SessionDir *SessionDir
}

// Cleanup removes the per-session temp dir.
func (p *SpawnPlan) Cleanup() error {
	if p.SessionDir == nil {
		return nil
	}
	return p.SessionDir.Remove()
}

// PrepareSpawn detects the shell, creates a session dir, writes the
// kubeconfig and shell-specific init, and returns an unstarted exec.Cmd.
func PrepareSpawn(opts Options, kubeconfig []byte) (*SpawnPlan, error) {
	d, err := Detect(opts.PathToShell)
	if err != nil {
		return nil, err
	}

	sd, err := newSessionDir(kubeconfig)
	if err != nil {
		return nil, err
	}

	// Build a clean env from os.Environ so user shell env (PATH etc.) carries
	// over but we can append our own KUBECONFIG/ZDOTDIR/etc.
	env := append([]string(nil), os.Environ()...)

	var cmd *exec.Cmd
	switch d.Kind {
	case KindBash:
		rcfile := filepath.Join(sd.Path, ".kapishrc")
		if err := os.WriteFile(rcfile, []byte(bashInit(opts, sd.KubeconfigPath)), 0o600); err != nil {
			_ = sd.Remove()
			return nil, fmt.Errorf("shell: write bash rcfile: %w", err)
		}
		cmd = exec.Command(d.Path, "--rcfile", rcfile)

	case KindZsh:
		zshrc := filepath.Join(sd.Path, ".zshrc")
		if err := os.WriteFile(zshrc, []byte(zshInit(opts, sd.KubeconfigPath)), 0o600); err != nil {
			_ = sd.Remove()
			return nil, fmt.Errorf("shell: write zshrc: %w", err)
		}
		env = append(env, "ZDOTDIR="+sd.Path)
		cmd = exec.Command(d.Path)

	case KindFish:
		// fish accepts the init script inline via --init-command=...
		init := fishInit(opts, sd.KubeconfigPath)
		cmd = exec.Command(d.Path, "--init-command="+init)

	default:
		_ = sd.Remove()
		return nil, fmt.Errorf("shell: unsupported kind %s", d.Kind)
	}

	cmd.Env = env

	return &SpawnPlan{Cmd: cmd, SessionDir: sd}, nil
}
```

- [ ] **Step 4: Run tests, expect pass**

Run: `go test ./internal/shell -v`
Expected: PASS for all `TestPrepareSpawn_*` plus existing.

- [ ] **Step 5: Commit**

```bash
git add internal/shell/spawn.go internal/shell/spawn_test.go
git commit -m "feat(shell): PrepareSpawn assembles per-shell exec.Cmd"
```

---

## Task 14: End-to-end shell spawn smoke test

**Files:**
- Create: `internal/shell/spawn_e2e_test.go`

- [ ] **Step 1: Write a smoke test that actually runs each shell**

Create `internal/shell/spawn_e2e_test.go`:

```go
package shell

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func haveShell(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not on PATH; skipping", name)
	}
	return p
}

func TestEnd2End_BashAppliesEnvAndAlias(t *testing.T) {
	bash := haveShell(t, "bash")

	plan, err := PrepareSpawn(Options{
		PathToShell: bash,
		Env:         map[string]string{"FOO": "bar"},
		Aliases:     map[string]string{"hi": "echo HELLO"},
	}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	// Run a non-interactive command in the rcfile-loaded bash.
	plan.Cmd.Args = append(plan.Cmd.Args, "-c", "echo $FOO; shopt -s expand_aliases; alias hi >/dev/null && hi || echo NO_HI")
	var out bytes.Buffer
	plan.Cmd.Stdout = &out
	plan.Cmd.Stderr = &out
	require.NoError(t, plan.Cmd.Run())

	assert.Contains(t, out.String(), "bar")
}

func TestEnd2End_ZshAppliesEnv(t *testing.T) {
	zsh := haveShell(t, "zsh")

	plan, err := PrepareSpawn(Options{
		PathToShell: zsh,
		Env:         map[string]string{"FOO": "bar"},
	}, []byte("# kc\n"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = plan.Cleanup() })

	plan.Cmd.Args = append(plan.Cmd.Args, "-c", "echo $FOO")
	var out bytes.Buffer
	plan.Cmd.Stdout = &out
	plan.Cmd.Stderr = &out
	require.NoError(t, plan.Cmd.Run())

	assert.Contains(t, out.String(), "bar")
}
```

> The test uses `t.Skip` if the shell isn't installed — keeps CI portable.

- [ ] **Step 2: Run smoke tests**

Run: `go test ./internal/shell -run TestEnd2End -v`
Expected: PASS (or skipped if shell isn't installed). On macOS bash 3.2's behavior with `-c` and a non-interactive shell: the `--rcfile` is sourced even with `-c`, so `$FOO` should propagate.

If a shell-specific assertion fails, investigate (don't disable). Note that `bash -c` runs in non-interactive mode and may treat aliases differently — that's why the bash test uses `shopt -s expand_aliases` before invoking the alias.

- [ ] **Step 3: Commit**

```bash
git add internal/shell/spawn_e2e_test.go
git commit -m "test(shell): end-to-end smoke against real bash/zsh"
```

---

## Task 15: Wire Plan 1 + Plan 2 — verify everything still builds

**Files:** none (verification only)

- [ ] **Step 1: Full test suite**

Run: `make test`
Expected: ALL packages green: `cmd/kapish`, `internal/capi`, `internal/config`, `internal/kapishlog`, `internal/shell`, `internal/version`.

- [ ] **Step 2: vet + tidy**

Run: `go vet ./... && go mod tidy`
Expected: clean; `go.mod`/`go.sum` no diff.

- [ ] **Step 3: Build the binary**

Run: `make build && ./bin/kapish version && ./bin/kapish --help`
Expected: still works (Plan 2 added libraries; the CLI surface didn't change yet).

- [ ] **Step 4: Confirm Plan 2 deliverables**

Both packages compile, are independently tested, and ready for the TUI (Plan 3) and Web backend (Plan 4) to consume.

If `go mod tidy` introduced changes, commit them:

```bash
git add go.mod go.sum
git commit -m "chore: tidy go.mod after Plan 2"
```

---

## Plan 2 exit criteria

- [ ] `make test` green across all 6 packages.
- [ ] `internal/capi` exposes: `NewClient(Options)`, `Client.ListClusters(ctx)`, `Client.WatchClusters(ctx)`, `Client.FetchKubeconfig(ctx, ns, name)`, `Cluster`, `Event`, `EventType`.
- [ ] `internal/shell` exposes: `Detect(string)`, `RenderPrompt(tmpl, tokens)`, `PrepareSpawn(Options, kubeconfig)`, `SpawnPlan{Cmd, SessionDir, Cleanup()}`, `Options`, `PromptTokens`, `Kind` constants.
- [ ] No package depends on cobra or any UI library.
- [ ] No integration test against a real kind cluster runs in `make test`. (We document setup separately if needed.)

When all boxes are checked, Plan 2 is done. Plan 3 (TUI) is next.
