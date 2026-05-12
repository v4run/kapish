# Web UI Settings + CAPI v1beta2 + CI Validation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the web-UI settings page (section-scoped panels, explicit per-section Save, `~`/`$HOME` expansion in working dir), center the "select a cluster" empty state with the real logo, migrate `internal/capi` from the deprecated CAPI `v1beta1` Cluster API to `v1beta2`, and add workflow linting to CI.

**Architecture:** Five mostly-independent changes. (1) Go `internal/capi`: swap import + GVR to `v1beta2`, adapt field access (`Status.Initialization.*`, value-typed `Spec.Topology`/`Spec.InfrastructureRef`), rename `FromV1Beta1`→`FromV1Beta2`. (2) Go `internal/shell`: new `expandCwd` (leading `~`→home, then `os.ExpandEnv`) + shared `cdLine` helper used by zsh/bash/fish. (3) React `SettingsView`: each left-nav item renders only its section; each section keeps a local draft with Save/Discard; theme stays instant. (4) React `EmptyState`/`App`: full-width centering + `<KapishMark>`. (5) `.github/workflows/ci.yml` + `Makefile`: `actionlint`.

**Tech Stack:** Go 1.26, `sigs.k8s.io/cluster-api` v1.13.1 (already in `go.mod`), `k8s.io/utils/ptr`, testify; React 18 + TypeScript + Vite + Tailwind; GitHub Actions, actionlint.

**Reference:** spec at `docs/superpowers/specs/2026-05-12-webui-and-capi-fixes-design.md`.

---

## Task 1: Migrate `internal/capi` to CAPI `v1beta2`

The package won't compile until source and test files are updated together, so this task changes all of them, then runs the suite. CAPI `v1.13.1` already provides `api/core/v1beta2`; no `go.mod` dependency is added (but `go mod tidy` may promote `k8s.io/utils` to a direct require — that's fine).

**Files:**
- Modify: `internal/capi/types.go`
- Modify: `internal/capi/list.go`
- Modify: `internal/capi/watch.go`
- Modify: `internal/capi/types_test.go`
- Modify: `internal/capi/list_test.go`
- Modify: `internal/capi/watch_test.go`

- [ ] **Step 1: Rewrite `internal/capi/types.go`**

```go
// Package capi wraps Cluster API types and exposes a small, focused API
// for kapish: list/watch CAPI Cluster CRDs and fetch workload kubeconfigs.
package capi

import (
	"strings"
	"time"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// Cluster is kapish's view of a CAPI Cluster. We don't expose the full
// v1beta2 type to consumers — TUI / Web only need a stable subset.
type Cluster struct {
	Name      string
	Namespace string

	Phase string

	ControlPlaneReady   bool
	InfrastructureReady bool

	K8sVersion string
	Provider   string

	CreationTimestamp time.Time
}

// FromV1Beta2 converts a CAPI v1beta2.Cluster into kapish's Cluster.
func FromV1Beta2(v *clusterv1.Cluster) Cluster {
	c := Cluster{
		Name:                v.Name,
		Namespace:           v.Namespace,
		Phase:               v.Status.Phase,
		ControlPlaneReady:   derefBool(v.Status.Initialization.ControlPlaneInitialized),
		InfrastructureReady: derefBool(v.Status.Initialization.InfrastructureProvisioned),
		CreationTimestamp:   v.CreationTimestamp.Time,
	}
	if v.Spec.Topology.Version != "" {
		c.K8sVersion = v.Spec.Topology.Version
	}
	if v.Spec.InfrastructureRef.Kind != "" {
		c.Provider = providerFromKind(v.Spec.InfrastructureRef.Kind)
	}
	return c
}

func derefBool(p *bool) bool { return p != nil && *p }

// providerFromKind extracts a short provider tag from the InfrastructureRef
// kind. Convention is <Provider>Cluster (AWSCluster, etc.). Unknown shapes
// return "".
func providerFromKind(kind string) string {
	if kind == "" {
		return ""
	}
	const suffix = "Cluster"
	if !strings.HasSuffix(kind, suffix) {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(kind, suffix))
}
```

- [ ] **Step 2: Update `internal/capi/list.go`**

Change the import line `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"` → `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"`. Change `clusterGVR`:

```go
// clusterGVR is the GroupVersionResource for cluster.x-k8s.io/v1beta2 Clusters.
var clusterGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta2",
	Resource: "clusters",
}
```

Replace both `out = append(out, FromV1Beta1(cl))` lines with `out = append(out, FromV1Beta2(cl))`.

- [ ] **Step 3: Update `internal/capi/watch.go`**

Change the import line to `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"`. In `clusterFromObject`, change `return FromV1Beta1(cl)` → `return FromV1Beta2(cl)`.

- [ ] **Step 4: Update `internal/capi/types_test.go`**

```go
package capi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

func TestFromV1Beta2_PopulatesFields(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "prod-eu-1",
			Namespace:         "prod",
			CreationTimestamp: metav1.NewTime(created),
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: clusterv1.ContractVersionedObjectReference{Kind: "AWSCluster"},
			Topology:          clusterv1.Topology{Version: "v1.30.2"},
		},
		Status: clusterv1.ClusterStatus{
			Phase: "Provisioned",
			Initialization: clusterv1.ClusterInitializationStatus{
				ControlPlaneInitialized:   ptr.To(true),
				InfrastructureProvisioned: ptr.To(true),
			},
		},
	}

	c := FromV1Beta2(v1)
	assert.Equal(t, "prod-eu-1", c.Name)
	assert.Equal(t, "prod", c.Namespace)
	assert.Equal(t, "Provisioned", c.Phase)
	assert.True(t, c.ControlPlaneReady)
	assert.True(t, c.InfrastructureReady)
	assert.Equal(t, "v1.30.2", c.K8sVersion)
	assert.Equal(t, "aws", c.Provider)
	assert.Equal(t, created, c.CreationTimestamp)
}

func TestFromV1Beta2_EmptyInfrastructureRefAndNilInit(t *testing.T) {
	v1 := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "no-infra"},
		Status:     clusterv1.ClusterStatus{Phase: "Pending"},
	}
	c := FromV1Beta2(v1)
	assert.Equal(t, "", c.Provider)
	assert.Equal(t, "", c.K8sVersion)
	assert.False(t, c.ControlPlaneReady)
	assert.False(t, c.InfrastructureReady)
}

func TestProviderFromKind(t *testing.T) {
	cases := map[string]string{
		"AWSCluster":       "aws",
		"GCPCluster":       "gcp",
		"AzureCluster":     "azure",
		"VSphereCluster":   "vsphere",
		"HetznerCluster":   "hetzner",
		"OpenStackCluster": "openstack",
		"":                 "",
	}
	for kind, want := range cases {
		require.Equal(t, want, providerFromKind(kind), "kind=%s", kind)
	}
}
```

- [ ] **Step 5: Update `internal/capi/list_test.go`**

Change the import `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"` → `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"`. Everything else in this file already only uses `clusterv1.Cluster`, `clusterv1.ClusterStatus{Phase: ...}`, `clusterv1.AddToScheme`, and `clusterv1.GroupVersion`, all of which exist in v1beta2 — no other change needed. (`newFakeCluster` lives here and is shared by `watch_test.go`.)

- [ ] **Step 6: Update `internal/capi/watch_test.go`**

Change the import `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"` → `clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"`. No other change.

- [ ] **Step 7: Tidy modules**

Run: `go mod tidy`
Expected: succeeds; `go.mod` may gain `k8s.io/utils` in a require block (still indirect or now direct) — that's fine. `git diff go.mod go.sum` should show no version downgrades.

- [ ] **Step 8: Build and test**

Run: `go build ./... && go vet ./... && go test ./internal/capi/... -count=1`
Expected: build/vet clean; tests PASS (`ok  github.com/v4run/kapish/internal/capi`).

- [ ] **Step 9: Full test suite**

Run: `make test`
Expected: all packages `ok`.

- [ ] **Step 10: Commit**

```bash
git add internal/capi go.mod go.sum
git commit -m "feat(capi): migrate Cluster API v1beta1 -> v1beta2

v1beta1 Cluster is deprecated; switch the dynamic GVR, types, and
conversion to core/v1beta2. ControlPlaneReady/InfrastructureReady now
read from Status.Initialization; Spec.Topology and Spec.InfrastructureRef
are value types in v1beta2.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 2: `~` / `$HOME` expansion in working directory (`internal/shell`)

TDD: write the test for `expandCwd` and `cdLine` first, watch it fail, implement, then refactor the three shell init functions to use `cdLine`.

**Files:**
- Create: `internal/shell/expand.go`
- Create: `internal/shell/expand_test.go`
- Modify: `internal/shell/zsh.go`
- Modify: `internal/shell/bash.go`
- Modify: `internal/shell/fish.go`

- [ ] **Step 1: Write the failing test — `internal/shell/expand_test.go`**

```go
package shell

import (
	"os"
	"testing"
)

func TestExpandCwd(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	t.Setenv("HOME", home) // keep os.ExpandEnv($HOME) consistent with UserHomeDir on this platform
	t.Setenv("KAPISH_TEST_DIR", "/tmp/k")

	cases := []struct{ in, want string }{
		{"", ""},
		{"~", home},
		{"~/work", home + "/work"},
		{"$HOME/work", home + "/work"},
		{"${HOME}/work", home + "/work"},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
		{"$KAPISH_TEST_DIR/sub", "/tmp/k/sub"},
		{"~unknownuser/x", "~unknownuser/x"}, // bare ~user form left unchanged
	}
	for _, c := range cases {
		if got := expandCwd(c.in); got != c.want {
			t.Errorf("expandCwd(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCdLine(t *testing.T) {
	if got := cdLine(""); got != "" {
		t.Errorf("cdLine(\"\") = %q, want empty", got)
	}
	home, _ := os.UserHomeDir()
	t.Setenv("HOME", home)
	if got := cdLine("~/with 'quote"); got != "cd "+posixSingleQuote(home+"/with 'quote")+"\n" {
		t.Errorf("cdLine quoting wrong: %q", got)
	}
}
```

- [ ] **Step 2: Run the test — verify it fails to compile**

Run: `go test ./internal/shell/ -run 'TestExpandCwd|TestCdLine' -v`
Expected: FAIL — `undefined: expandCwd`, `undefined: cdLine`.

- [ ] **Step 3: Implement — `internal/shell/expand.go`**

```go
package shell

import (
	"os"
	"strings"
)

// expandCwd expands a leading ~ or ~/ to the user's home directory, then
// expands $VAR / ${VAR} via the process environment. An empty string is
// returned unchanged, as is a bare ~user form (rare; not worth an os/user
// lookup). If the home directory can't be determined, the ~ is left as-is.
func expandCwd(p string) string {
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	return os.ExpandEnv(p)
}

// cdLine returns a shell line that cds into the (expanded) working directory,
// or "" when cwd is empty. The same `cd 'dir'` syntax works in zsh, bash and
// fish.
func cdLine(cwd string) string {
	cwd = expandCwd(cwd)
	if cwd == "" {
		return ""
	}
	return "cd " + posixSingleQuote(cwd) + "\n"
}
```

- [ ] **Step 4: Run the test — verify it passes**

Run: `go test ./internal/shell/ -run 'TestExpandCwd|TestCdLine' -v`
Expected: PASS.

- [ ] **Step 5: Refactor `zsh.go` to use `cdLine`**

Replace:
```go
	if opts.Cwd != "" {
		b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n")
	}
```
with:
```go
	b.WriteString(cdLine(opts.Cwd))
```
(`b.WriteString("")` is a no-op, so no guard needed.)

- [ ] **Step 6: Refactor `bash.go` to use `cdLine`**

Same replacement as Step 5 in `bashInit`.

- [ ] **Step 7: Refactor `fish.go` to use `cdLine`**

Same replacement as Step 5 in `fishInit`.

- [ ] **Step 8: Run shell package tests**

Run: `go test ./internal/shell/ -count=1`
Expected: PASS (existing rcfile-generation tests still pass — output is byte-identical when `opts.Cwd` is empty or already absolute).

- [ ] **Step 9: Commit**

```bash
git add internal/shell
git commit -m "feat(shell): expand ~ and \$HOME in working directory

New expandCwd (leading ~ -> home, then os.ExpandEnv) and a shared cdLine
helper used by the zsh/bash/fish init generators. Config still stores the
literal value (e.g. ~/work) so it stays portable.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 3: Section-scoped settings panel with per-section Save

The current `SettingsSectionForm` renders all fields regardless of the active tab, and `SettingsView.applyForm` persists on every keystroke. Replace both: `SettingsView` renders one section component per tab; each section component keeps a local draft seeded from the loaded config, with a `Save`/`Discard` bar that's disabled until dirty. Theme stays instant-apply.

**Files:**
- Create: `internal/web/frontend/src/ui/SettingsSections.tsx`
- Modify: `internal/web/frontend/src/SettingsView.tsx`
- Delete: `internal/web/frontend/src/ui/SettingsSectionForm.tsx`
- Modify: `internal/web/frontend/src/ui/Field.tsx` (working-dir hint text only — actually the hint is passed in by the section, so no change here; left out)

> Note: there is no JS test harness in this repo. Verification is `npm run build` (TypeScript type-check via `tsc -b`) plus the manual click-through in Step 6.

- [ ] **Step 1: Create `internal/web/frontend/src/ui/SettingsSections.tsx`**

```tsx
import * as React from 'react';
import { TextField, Select } from './Field';
import { KVList, KV } from './KVList';
import { Button } from './Button';
import type { KapishConfig } from '../api/types';

// v1 supported shells only (handoff also listed nu/pwsh; deferred).
export type Shell = 'zsh' | 'bash' | 'fish';
const SHELLS: Shell[] = ['zsh', 'bash', 'fish'];
const DEFAULT_PROMPT = '[{cluster}] ';

function kvFromRecord(r?: Record<string, string>): KV[] {
  return Object.entries(r ?? {}).map(([k, v]) => ({ k, v }));
}
function recordFromKV(items: KV[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const { k, v } of items) if (k) out[k] = v;
  return out;
}
function basename(p?: string): string {
  return (p ?? '').split('/').pop() ?? '';
}
function shellFromCommand(cmd?: string): Shell {
  const b = basename(cmd);
  return (SHELLS as string[]).includes(b) ? (b as Shell) : 'zsh';
}
// Keep the existing full path when the picked shell still matches its basename;
// otherwise write the bare shell name.
function commandForShell(shell: Shell, cmd?: string): string {
  return basename(cmd) === shell ? (cmd ?? '') : shell;
}

// onSave applies a patch to the latest config and PUTs it; it throws on error.
type OnSave = (patch: (c: KapishConfig) => KapishConfig) => Promise<void>;

function useSectionSave(onSave: OnSave) {
  const [saving, setSaving] = React.useState(false);
  const [savedAt, setSavedAt] = React.useState<number | null>(null);
  const [err, setErr] = React.useState<string | null>(null);
  const run = async (patch: (c: KapishConfig) => KapishConfig) => {
    setSaving(true);
    setErr(null);
    try {
      await onSave(patch);
      setSavedAt(Date.now());
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };
  return { saving, savedAt, err, run };
}

function SaveBar({ dirty, saving, savedAt, err, onSave, onDiscard }: {
  dirty: boolean; saving: boolean; savedAt: number | null; err: string | null; onSave: () => void; onDiscard: () => void;
}) {
  return (
    <div className="flex items-center gap-3 pt-1">
      <Button variant="primary" size="sm" disabled={!dirty || saving} onClick={onSave}>Save</Button>
      <Button variant="secondary" size="sm" disabled={!dirty || saving} onClick={onDiscard}>Discard</Button>
      {saving && <span className="text-2xs text-muted">saving…</span>}
      {!saving && dirty && <span className="text-2xs text-dim">unsaved changes</span>}
      {!saving && !dirty && savedAt && <span className="text-2xs text-success">saved</span>}
      {err && <span className="text-2xs text-error">{err}</span>}
    </div>
  );
}

export function ShellPromptSection({ cfg, onSave }: { cfg: KapishConfig; onSave: OnSave }) {
  const baseShell = shellFromCommand(cfg.shell.command);
  const basePrompt = cfg.shell.prompt ?? DEFAULT_PROMPT;
  const [shell, setShell] = React.useState<Shell>(baseShell);
  const [prompt, setPrompt] = React.useState(basePrompt);
  React.useEffect(() => { setShell(baseShell); setPrompt(basePrompt); }, [cfg]); // re-seed after a save elsewhere
  const dirty = shell !== baseShell || prompt !== basePrompt;
  const ss = useSectionSave(onSave);
  return (
    <div className="grid gap-5 max-w-3xl">
      <Select label="Shell" value={shell} options={SHELLS} onChange={setShell} />
      <TextField label="Prompt template" mono value={prompt} onChange={setPrompt} hint="tokens: {cluster} {ns} {provider} {ctx} {time}" />
      <SaveBar dirty={dirty} saving={ss.saving} savedAt={ss.savedAt} err={ss.err}
        onDiscard={() => { setShell(baseShell); setPrompt(basePrompt); }}
        onSave={() => ss.run((c) => ({ ...c, shell: { ...c.shell, command: commandForShell(shell, c.shell.command), prompt } }))} />
    </div>
  );
}

export function EnvVarsSection({ cfg, onSave }: { cfg: KapishConfig; onSave: OnSave }) {
  const base = kvFromRecord(cfg.shell.env);
  const [items, setItems] = React.useState<KV[]>(base);
  React.useEffect(() => { setItems(kvFromRecord(cfg.shell.env)); }, [cfg]);
  const dirty = JSON.stringify(recordFromKV(items)) !== JSON.stringify(recordFromKV(base));
  const ss = useSectionSave(onSave);
  return (
    <div className="grid gap-5 max-w-3xl">
      <KVList label="Environment variables" hint={`${items.length} active`} items={items} onChange={setItems} keyPlaceholder="KUBE_EDITOR" valuePlaceholder="nvim" />
      <SaveBar dirty={dirty} saving={ss.saving} savedAt={ss.savedAt} err={ss.err}
        onDiscard={() => setItems(kvFromRecord(cfg.shell.env))}
        onSave={() => ss.run((c) => ({ ...c, shell: { ...c.shell, env: recordFromKV(items) } }))} />
    </div>
  );
}

export function AliasesSection({ cfg, onSave }: { cfg: KapishConfig; onSave: OnSave }) {
  const base = kvFromRecord(cfg.shell.aliases);
  const [items, setItems] = React.useState<KV[]>(base);
  React.useEffect(() => { setItems(kvFromRecord(cfg.shell.aliases)); }, [cfg]);
  const dirty = JSON.stringify(recordFromKV(items)) !== JSON.stringify(recordFromKV(base));
  const ss = useSectionSave(onSave);
  return (
    <div className="grid gap-5 max-w-3xl">
      <KVList label="Aliases" hint={`${items.length} active`} items={items} onChange={setItems} keyPlaceholder="k" valuePlaceholder="kubectl" />
      <SaveBar dirty={dirty} saving={ss.saving} savedAt={ss.savedAt} err={ss.err}
        onDiscard={() => setItems(kvFromRecord(cfg.shell.aliases))}
        onSave={() => ss.run((c) => ({ ...c, shell: { ...c.shell, aliases: recordFromKV(items) } }))} />
    </div>
  );
}

export function WorkingDirSection({ cfg, onSave }: { cfg: KapishConfig; onSave: OnSave }) {
  const base = cfg.shell.cwd ?? '';
  const [cwd, setCwd] = React.useState(base);
  React.useEffect(() => { setCwd(cfg.shell.cwd ?? ''); }, [cfg]);
  const dirty = cwd !== base;
  const ss = useSectionSave(onSave);
  return (
    <div className="grid gap-5 max-w-3xl">
      <TextField label="Working directory" mono value={cwd} onChange={setCwd} hint="~ and $HOME are expanded; empty = inherit" />
      <SaveBar dirty={dirty} saving={ss.saving} savedAt={ss.savedAt} err={ss.err}
        onDiscard={() => setCwd(cfg.shell.cwd ?? '')}
        onSave={() => ss.run((c) => ({ ...c, shell: { ...c.shell, cwd } }))} />
    </div>
  );
}
```

- [ ] **Step 2: Rewrite `internal/web/frontend/src/SettingsView.tsx`**

```tsx
import * as React from 'react';
import { AppHeader } from './ui/AppHeader';
import { SettingsTabs, Tab } from './ui/SettingsTabs';
import { ShellPromptSection, EnvVarsSection, AliasesSection, WorkingDirSection } from './ui/SettingsSections';
import { Select } from './ui/Field';
import { Button } from './ui/Button';
import { getConfig, putConfig } from './api/client';
import type { KapishConfig } from './api/types';

const TABS: Tab[] = [
  { id: 'shell', label: 'Shell & prompt' },
  { id: 'env', label: 'Env vars' },
  { id: 'aliases', label: 'Aliases' },
  { id: 'cwd', label: 'Working dir' },
  { id: 'theme', label: 'Theme' },
];

export function SettingsView({ mgmtCluster, onClose }: { mgmtCluster: string; onClose: () => void }) {
  const [cfg, setCfg] = React.useState<KapishConfig | null>(null);
  const [tab, setTab] = React.useState('shell');
  const [err, setErr] = React.useState<string | null>(null);

  React.useEffect(() => {
    getConfig().then(setCfg).catch((e) => setErr(e instanceof Error ? e.message : String(e)));
  }, []);

  // Apply a patch to the latest config and persist. Throws on failure so the
  // calling section can surface it; updates cfg on success so other sections
  // re-seed from the new value. cfgRef keeps the callback identity stable while
  // always patching the freshest config.
  const cfgRef = React.useRef<KapishConfig | null>(null);
  React.useEffect(() => { cfgRef.current = cfg; }, [cfg]);
  const saveSection = React.useCallback(async (patch: (c: KapishConfig) => KapishConfig) => {
    const cur = cfgRef.current;
    if (!cur) return;
    const next = patch(cur);
    await putConfig(next);
    setCfg(next);
  }, []);

  const toggleTheme = () => {
    if (!cfg) return;
    const theme = cfg.ui.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    const next: KapishConfig = { ...cfg, ui: { ...cfg.ui, theme } };
    putConfig(next).then(() => setCfg(next)).catch((e) => setErr(e instanceof Error ? e.message : String(e)));
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-bg text-text">
      <AppHeader mgmtCluster={mgmtCluster} onSettings={onClose} />
      <div className="flex-1 min-h-0 flex">
        <SettingsTabs tabs={TABS} active={tab} onSelect={setTab} />
        <main className="flex-1 min-w-0 overflow-y-auto p-6">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-semibold">{TABS.find((t) => t.id === tab)?.label}</h1>
            <Button variant="secondary" size="sm" onClick={onClose}>Close</Button>
          </div>
          {err && <div className="mb-4 text-sm text-error">{err}</div>}
          {!cfg && !err && <div className="text-sm text-muted">Loading…</div>}
          {cfg && tab === 'shell' && <ShellPromptSection cfg={cfg} onSave={saveSection} />}
          {cfg && tab === 'env' && <EnvVarsSection cfg={cfg} onSave={saveSection} />}
          {cfg && tab === 'aliases' && <AliasesSection cfg={cfg} onSave={saveSection} />}
          {cfg && tab === 'cwd' && <WorkingDirSection cfg={cfg} onSave={saveSection} />}
          {cfg && tab === 'theme' && (
            <Select label="Theme" value={cfg.ui.theme as 'dark' | 'light'} options={['dark', 'light']} onChange={() => toggleTheme()} />
          )}
        </main>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Delete the old form**

Run: `git rm internal/web/frontend/src/ui/SettingsSectionForm.tsx`
Then `grep -rn SettingsSectionForm internal/web/frontend/src` — expect no matches.

- [ ] **Step 4: Type-check / build the frontend**

Run: `cd internal/web/frontend && npm run build`
Expected: `tsc -b` passes with no errors; Vite writes `dist/`.

- [ ] **Step 5: Refresh the committed `dist/`**

The repo commits the built frontend (`internal/web/frontend/dist/`, embedded via `internal/web/embed.go`). `npm run build` regenerated it. Stage it.

Run: `git status --porcelain internal/web/frontend/dist`
Expected: shows modified/new `dist/assets/*` and `dist/index.html`.

- [ ] **Step 6: Manual smoke test**

Run: `go run ./cmd/kapish web --dev` (or `go run ./cmd/kapish web` after the build) and open the settings page. Verify:
- Each left-nav item shows ONLY its fields (Shell & prompt / Env vars / Aliases / Working dir / Theme).
- Editing a field enables Save + Discard and shows "unsaved changes".
- Discard reverts the field; Save persists, shows "saved", and a reload shows the persisted value.
- Working dir hint reads "~ and $HOME are expanded; empty = inherit".
- Theme select still toggles immediately (no Save bar).

- [ ] **Step 7: Commit**

```bash
git add internal/web/frontend/src internal/web/frontend/dist
git commit -m "feat(web): section-scoped settings with per-section Save

Each settings tab now renders only its own fields; each section keeps a
local draft with Save/Discard, replacing save-on-keystroke. Working dir
hint mentions ~/\$HOME expansion. Theme stays instant-apply.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 4: Center the "select a cluster" empty state and use the real logo

`SelectClusterEmpty` is rendered inside `<div className="flex-1 flex">…</div>` in `App.tsx`; `EmptyState`'s root has no width so as a flex item it shrinks to content width and sits flush-left. It also uses a bespoke inline SVG. Give the pane full width to the empty state and swap the SVG for `<KapishMark>`.

**Files:**
- Modify: `internal/web/frontend/src/App.tsx`
- Modify: `internal/web/frontend/src/ui/EmptyState.tsx`
- Modify: `internal/web/frontend/src/ui/EmptyState.tsx` (import)

- [ ] **Step 1: Fix the wrapper in `App.tsx`**

Find:
```tsx
        ) : (
          <div className="flex-1 flex"><SelectClusterEmpty /></div>
        )}
```
Replace with:
```tsx
        ) : (
          <div className="flex-1"><SelectClusterEmpty /></div>
        )}
```
(`EmptyState`'s root is `h-full flex flex-col items-center justify-center … text-center`; with the wrapper now sized by `flex-1` and `h-full` giving it height, `items-center`/`text-center` take effect.)

- [ ] **Step 2: Use `KapishMark` in `SelectClusterEmpty` (`EmptyState.tsx`)**

Add the import at the top of `internal/web/frontend/src/ui/EmptyState.tsx`:
```tsx
import { KapishMark } from '../brand/KapishMark';
```
Replace `SelectClusterEmpty`'s `icon` prop:
```tsx
export const SelectClusterEmpty = () => (
  <EmptyState
    icon={<KapishMark size={36} />}
    title="Select a cluster to start a shell"
    body="Pick any cluster from the list. kapish will fetch its kubeconfig and spawn your configured shell." />
);
```
Note: `EmptyState` wraps `icon` in `<div className="text-muted mb-3">`. `KapishMark` uses `accent="currentColor"` for the vertical stroke/dots, so it would render muted-grey; that's acceptable and matches the muted empty-state aesthetic, while the violet diagonal strokes keep their color. Leave the `text-muted` wrapper as-is (don't special-case `EmptyState`).

- [ ] **Step 3: Build the frontend**

Run: `cd internal/web/frontend && npm run build`
Expected: `tsc -b` passes; `dist/` regenerated.

- [ ] **Step 4: Manual check**

Run kapish web, leave no cluster selected: the empty state is horizontally centered in the right pane and shows the kapish mark above the message.

- [ ] **Step 5: Commit**

```bash
git add internal/web/frontend/src internal/web/frontend/dist
git commit -m "fix(web): center the empty cluster state, use the kapish mark

The right-pane empty state was flush-left because EmptyState had no width
as a flex item; the wrapper now sizes it. Swapped the ad-hoc SVG for the
shared KapishMark.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 5: Validate GitHub Actions workflows in CI (actionlint)

Add `actionlint` to `ci.yml` and a `make lint-actions` target wired into `make lint`.

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `Makefile`

- [ ] **Step 1: Add the actionlint step to `.github/workflows/ci.yml`**

In the `test` job's `steps:`, after the `Test` step, append:
```yaml
      - name: Lint workflows
        uses: raven-actions/actionlint@v2
```

- [ ] **Step 2: Add `lint-actions` to the `Makefile` and wire into `lint`**

Replace the current `lint` target:
```makefile
lint:
	$(GO) vet ./...
```
with:
```makefile
lint: lint-actions
	$(GO) vet ./...

lint-actions:
	$(GO) run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/*.yml
```
Add `lint-actions` to the `.PHONY` line that currently reads `.PHONY: all build install test lint fmt tidy clean` → `.PHONY: all build install test lint lint-actions fmt tidy clean`.

- [ ] **Step 3: Run actionlint locally**

Run: `make lint-actions`
Expected: downloads actionlint on first run, then exits 0 with no findings. If it reports issues in `build.yml`/`ci.yml`, fix them (they are real workflow bugs) before continuing.

- [ ] **Step 4: Run `make lint`**

Run: `make lint`
Expected: `lint-actions` passes, then `go vet ./...` passes.

- [ ] **Step 5: Commit**

```bash
git add .github/workflows/ci.yml Makefile
git commit -m "ci: lint workflows with actionlint

New 'Lint workflows' step in ci.yml and a 'make lint-actions' target
(wired into 'make lint') so workflow syntax errors are caught.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 6: Final verification

- [ ] **Step 1: Full Go suite + vet + build**

Run: `go build ./... && go vet ./... && make test`
Expected: build clean, vet clean, all packages `ok`.

- [ ] **Step 2: goimports check (matches CI)**

Run: `go run golang.org/x/tools/cmd/goimports@latest -l $(git ls-files '*.go')`
Expected: no output. If any file is listed, run the same command with `-w` and amend the relevant commit.

- [ ] **Step 3: Frontend build**

Run: `make frontend`
Expected: `npm install` + `npm run build` succeed; `git status` shows `dist/` already committed (clean) — if not, `git add` it and amend the Task 3/4 commit.

- [ ] **Step 4: actionlint**

Run: `make lint-actions`
Expected: exits 0.

- [ ] **Step 5: Confirm the original warnings are gone**

- `grep -rn "v1beta1" $(git ls-files 'internal/**/*.go')` → no matches (the deprecation source is removed).
- Spot-check: `grep -rn "applyForm\|save on key" internal/web/frontend/src` → no matches (save-on-keystroke removed).

---

## Self-Review Notes (for the implementer)

- **Spec coverage:** §1 → Task 3; §2 (`~`/`$HOME`) → Task 2; §3 (mgmt selector) → no code (intentional); §4 (empty state) → Task 4; §5 (v1beta2) → Task 1; §6 (workflow lint) → Task 5.
- **`ClusterInitializationStatus` field names** are `ControlPlaneInitialized` and `InfrastructureProvisioned` (both `*bool`); `Status.Initialization` is a value (not pointer). `Spec.Topology` is a value with a `Version string` field; `Spec.InfrastructureRef` is a value `ContractVersionedObjectReference` with a `Kind string` field. Verified against `sigs.k8s.io/cluster-api@v1.13.1/api/core/v1beta2/cluster_types.go`.
- **`FromV1Beta2`** is the only renamed symbol; callers are in `list.go` (×2) and `watch.go` (×1).
- **`cdLine`/`expandCwd`** are package-private (`internal/shell`); the only call sites are the three `*Init` functions.
- **Frontend has no test runner** — `npm run build` (which runs `tsc -b`) is the type-check gate; behavior is checked manually per Task 3 Step 6 / Task 4 Step 4.
