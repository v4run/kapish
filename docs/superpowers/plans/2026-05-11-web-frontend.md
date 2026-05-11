# Web Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the React + Tailwind + xterm.js frontend at `internal/web/frontend/`, drop in the Claude Design components from `docs/superpowers/design/web-ui-handoff.md`, wire them to the Plan 4 API (REST + SSE + the WebSocket-PTY framing protocol), build with Vite, and commit the build output to `internal/web/frontend/dist/` so `//go:embed` (already wired in Plan 4) ships the real UI under `go install`. After this plan, `kapish serve` opens a working browser UI: cluster list with live updates, click a cluster → terminal, settings page, mgmt picker.

**Architecture:**
- Vite project (TS + React 18 + Tailwind). Single-route SPA: `App` conditionally renders the main view (header + cluster sidebar + terminal pane) or the settings view; a mgmt-picker dropdown overlays the header. No router.
- `src/api/client.ts` — typed `fetch` wrappers (`getClusters`, `getConfig`, `putConfig`, `getMgmts`, `putMgmtsCurrent`, `createSession`). `src/api/stream.ts` — an `EventSource` subscriber for `/api/v1/clusters/stream` (on `sync` → re-fetch the snapshot; on `cluster` → patch a local list). `src/api/ws.ts` — the framing helper: `0x00`+bytes = data (both directions), `0x01`+JSON `{cols,rows}` = resize (client→server), `0x02` = ping/pong.
- `src/components/TerminalSession.tsx` owns one xterm.js `Terminal` + `FitAddon` + the WebSocket: on mount it `createSession({namespace,cluster})`, opens `ws://…/api/v1/sessions/<id>/ws?token=<tok>` with `binaryType="arraybuffer"`, pipes `term.onData → 0x00 frame`, `fitAddon`/resize → `0x01 frame`, `ws.onmessage` (`data[0]===0x00 → term.write(data.subarray(1))`), pings every 30s; on unmount closes the WS and disposes the terminal.
- `src/components/MgmtPicker.tsx` — dropdown anchored to the header chip; `getMgmts()` on open, `putMgmtsCurrent({name})` on select, then bubbles an `onSwitched` so `App` re-fetches clusters.
- `kapish serve --dev` (Plan 4 added the flag as a no-op) gets wired here: in dev mode the Go server proxies `/` to `vite dev`'s port; the frontend dev server proxies `/api` (with `ws: true` for `/api/v1/sessions/*/ws`) back to the Go server.
- Build: `make frontend` (or `pnpm --dir internal/web/frontend build`) → `internal/web/frontend/dist/`. The dist output is committed (it's small) so `go install github.com/v4run/kapish/cmd/kapish@latest` ships the UI without a Node toolchain.

**Tech Stack:**
- `react` 18, `react-dom` — UI
- `xterm` (`@xterm/xterm`) + `@xterm/addon-fit` — terminal
- `vite` + `@vitejs/plugin-react` — build/dev
- `tailwindcss` + `postcss` + `autoprefixer` — styling
- `@tanstack/react-query` — cluster list fetching/caching (optional; plain `useEffect` is fine too — pick whichever is simpler)
- TypeScript
- No router, no UI component library (everything in-house per the handoff)
- (Optional, Task 13) `@playwright/test` for one E2E smoke

**Reference:** `docs/superpowers/design/web-ui-handoff.md` has the verbatim source for tokens, brand, icons, and all UI components. Plan tasks below say "copy component X from the handoff" — read that file for the code.

**End-state:** `make build && ./bin/kapish serve --port 0 --no-open` serves a real React UI at `/`; opening it in a browser shows the cluster list (live), clicking a cluster opens an in-browser shell, the settings page reads/writes config, and the mgmt chip switches management clusters. `go install` ships the same.

---

## Task 1: Vite project scaffold

**Files (all under `internal/web/frontend/`):**
- Create: `package.json`, `vite.config.ts`, `tsconfig.json`, `tsconfig.node.json`, `postcss.config.js`, `tailwind.config.js`, `index.html`, `src/main.tsx`, `src/styles/tokens.css`, `src/vite-env.d.ts`
- Modify: `.gitignore` (root) — add `internal/web/frontend/node_modules/`
- Modify: `Makefile` — add `frontend` target

- [ ] **Step 1: package.json**

Create `internal/web/frontend/package.json`:
```json
{
  "name": "kapish-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "@tanstack/react-query": "^5.59.0",
    "@xterm/addon-fit": "^0.10.0",
    "@xterm/xterm": "^5.5.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0",
    "tailwindcss": "^3.4.0",
    "typescript": "^5.6.0",
    "vite": "^5.4.0"
  }
}
```

- [ ] **Step 2: vite.config.ts** — `internal/web/frontend/vite.config.ts`:
```ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In `kapish serve --dev`, the Go server proxies `/` here; this dev server
// proxies /api (including WebSocket upgrades) back to the Go server. The Go
// server's port is passed via VITE_KAPISH_API (default localhost:0 won't work,
// so --dev passes the real port through an env var).
const apiTarget = process.env.VITE_KAPISH_API || 'http://127.0.0.1:8765';

export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    proxy: {
      '/api': { target: apiTarget, changeOrigin: false, ws: true },
    },
  },
});
```

- [ ] **Step 3: tsconfig.json** + **tsconfig.node.json** — standard Vite-React TS configs:

`internal/web/frontend/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

`internal/web/frontend/tsconfig.node.json`:
```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "noEmit": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 4: postcss.config.js** — `internal/web/frontend/postcss.config.js`:
```js
export default { plugins: { tailwindcss: {}, autoprefixer: {} } };
```

- [ ] **Step 5: tailwind.config.js** — copy the `theme.extend` block verbatim from the handoff (`docs/superpowers/design/web-ui-handoff.md` §1). Use CommonJS (`module.exports`) as the handoff shows, OR ESM (`export default`) — Tailwind accepts both; match the handoff's `module.exports` form but if Vite complains about CJS in an ESM package (`"type": "module"`), rename to `tailwind.config.cjs` and keep `module.exports`. (Safest: name it `tailwind.config.cjs` with `module.exports`.)

- [ ] **Step 6: index.html** — `internal/web/frontend/index.html`:
```html
<!doctype html>
<html lang="en" data-theme="dark">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>kapish</title>
  </head>
  <body class="bg-bg text-text">
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 7: src/main.tsx** — `internal/web/frontend/src/main.tsx`:
```tsx
import React from 'react';
import ReactDOM from 'react-dom/client';
import './styles/tokens.css';
import '@xterm/xterm/css/xterm.css';
import App from './App';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
```

- [ ] **Step 8: src/styles/tokens.css** — copy verbatim from the handoff §1 (the CSS custom properties block). Also append a Tailwind directive header at the very top:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```
(So the file is: `@tailwind` directives, then the `:root` token blocks, then the `html`/`code`/`@keyframes` rules from the handoff.)

- [ ] **Step 9: src/vite-env.d.ts** — `/// <reference types="vite/client" />`

- [ ] **Step 10: src/App.tsx (temporary)** — a minimal placeholder so the build compiles:
```tsx
export default function App() {
  return <div className="p-6 text-text">kapish — frontend scaffold (wired in later tasks)</div>;
}
```

- [ ] **Step 11: .gitignore** — add a line `internal/web/frontend/node_modules/` to the repo root `.gitignore` (so npm/pnpm install dirs aren't committed). Note `internal/web/frontend/dist/` is intentionally NOT ignored — we commit the build output.

- [ ] **Step 12: Makefile** — add a `frontend` target:
```makefile
.PHONY: frontend
frontend:
	cd internal/web/frontend && npm install && npm run build
```
(Use `npm` for portability; `pnpm` is fine too if available — keep it to `npm` for the plan.) Also make `build` depend on nothing new (the committed `dist/` is what `go build` embeds; `make frontend` is run manually before commits that change the UI).

- [ ] **Step 13: Install + build**

Run:
```sh
cd internal/web/frontend
npm install
npm run build
```
Expected: `npm install` succeeds; `npm run build` produces `internal/web/frontend/dist/{index.html,assets/...}`. (This overwrites the Plan 4 placeholder `dist/index.html` — that's intended; the scaffold App renders the placeholder text for now.)

- [ ] **Step 14: Verify Go still builds + embeds**

Run from repo root:
```sh
go build ./... && go test ./... -count=1
make build && ./bin/kapish serve --port 0 --no-open --kubeconfig /tmp/nope --context nope 2>&1 | head -3
```
The serve command will still error on the bad kubeconfig (clean error, not a panic) — that's fine; the point is `go build` embeds the new `dist/` without complaint. `go test ./...` should still be green (the `internal/web/embed_test.go` `TestServesIndexAtRoot` now serves the Vite-built index, which still contains "kapish" — assert still passes).

- [ ] **Step 15: Commit**
```bash
git add internal/web/frontend/ Makefile .gitignore
git commit -m "feat(web): scaffold Vite + React + Tailwind frontend"
```

---

## Task 2: Design tokens, brand, icons — drop in from handoff

**Files (under `internal/web/frontend/src/`):**
- Verify/update: `styles/tokens.css` (done in Task 1 — confirm it matches handoff §1)
- Create: `brand/KapishMark.tsx`, `brand/KapishLockup.tsx`, `icons/Icon.tsx`

- [ ] **Step 1:** Create `src/brand/KapishMark.tsx` — verbatim from handoff §2.
- [ ] **Step 2:** Create `src/brand/KapishLockup.tsx` — verbatim from handoff §2 (exports `KapishWordmark` + `KapishLockup`).
- [ ] **Step 3:** Create `src/icons/Icon.tsx` — verbatim from handoff §2 (the `I` base + all `Icon*` exports). Note: the handoff uses `(p: any)` for the icon props — fine, but if `noUnusedParameters`/strict TS complains, type it as `React.SVGProps<SVGSVGElement> & { size?: number }` instead of `any`. Try `any` first; tighten only if `tsc` fails.
- [ ] **Step 4:** `cd internal/web/frontend && npx tsc -b --noEmit` (or `npm run build`) — compiles clean.
- [ ] **Step 5: Commit**
```bash
git add internal/web/frontend/src/brand internal/web/frontend/src/icons internal/web/frontend/src/styles
git commit -m "feat(web): brand assets, icon set, design tokens"
```

---

## Task 3: UI primitives — drop in from handoff

**Files (under `internal/web/frontend/src/ui/`):**
- Create: `Button.tsx`, `FilterInput.tsx`, `PhaseChip.tsx`, `EmptyState.tsx`, `ConfirmDialog.tsx`, `Toast.tsx`, `Field.tsx`, `KVList.tsx`, `ErrorBanner.tsx`, `ClusterListSkeleton.tsx`

- [ ] **Step 1:** Create each file verbatim from handoff §3 / §5. Note these handoff-vs-v1 adjustments (already reflected in the handoff doc): `PhaseChip` takes `phase: string` (tolerates unknown phases); `ClusterListRow`'s `PROVIDER_TINT` lookup tolerates unknown providers; `SettingsSectionForm`'s `Shell` type is `'zsh' | 'bash' | 'fish'` (no `nu`/`pwsh`).
- [ ] **Step 2:** `cd internal/web/frontend && npm run build` — compiles clean (the components are self-contained; they only import from `../brand`, `../icons`, and each other).
- [ ] **Step 3: Commit**
```bash
git add internal/web/frontend/src/ui/Button.tsx internal/web/frontend/src/ui/FilterInput.tsx internal/web/frontend/src/ui/PhaseChip.tsx internal/web/frontend/src/ui/EmptyState.tsx internal/web/frontend/src/ui/ConfirmDialog.tsx internal/web/frontend/src/ui/Toast.tsx internal/web/frontend/src/ui/Field.tsx internal/web/frontend/src/ui/KVList.tsx internal/web/frontend/src/ui/ErrorBanner.tsx internal/web/frontend/src/ui/ClusterListSkeleton.tsx
git commit -m "feat(web): UI primitives (Button, FilterInput, PhaseChip, dialogs, etc.)"
```

---

## Task 4: Composite components — AppHeader, ClusterListRow, TerminalPanel, SettingsTabs, SettingsSectionForm

**Files (under `internal/web/frontend/src/ui/`):**
- Create: `AppHeader.tsx`, `ClusterListRow.tsx`, `TerminalPanel.tsx`, `SettingsTabs.tsx`, `SettingsSectionForm.tsx`

- [ ] **Step 1:** Create each verbatim from handoff §3.
- [ ] **Step 2:** `cd internal/web/frontend && npm run build` — compiles clean.
- [ ] **Step 3: Commit**
```bash
git add internal/web/frontend/src/ui/AppHeader.tsx internal/web/frontend/src/ui/ClusterListRow.tsx internal/web/frontend/src/ui/TerminalPanel.tsx internal/web/frontend/src/ui/SettingsTabs.tsx internal/web/frontend/src/ui/SettingsSectionForm.tsx
git commit -m "feat(web): composite components (AppHeader, ClusterListRow, TerminalPanel, settings)"
```

---

## Task 5: API client — typed fetch wrappers

**Files:**
- Create: `internal/web/frontend/src/api/types.ts`, `internal/web/frontend/src/api/client.ts`

- [ ] **Step 1:** `src/api/types.ts` — TypeScript shapes matching the Plan 4 endpoints:
```ts
export interface Cluster {
  name: string;
  namespace: string;
  phase: string;
  controlPlaneReady: boolean;
  infrastructureReady: boolean;
  version: string;
  provider: string;
  ageSeconds: number;
}
export interface MgmtEntry { name: string; context?: string; namespace?: string }
export interface Mgmts { current: string; entries: MgmtEntry[] }
export interface CreateSessionResp { sessionId: string; wsUrl: string; wsToken: string }
// Config mirrors kconfig.Config's JSON (lowercase keys).
export interface KapishConfig {
  managementClusters: { current?: string; entries?: MgmtEntry[] };
  shell: { command?: string; cwd?: string; env?: Record<string, string>; aliases?: Record<string, string>; prompt?: string };
  ui: { theme: string; refreshIntervalSec: number; oneShot: boolean };
  web: { defaultPort: number; openBrowser: boolean; bindAddr: string };
}
```

- [ ] **Step 2:** `src/api/client.ts`:
```ts
import type { Cluster, Mgmts, CreateSessionResp, KapishConfig } from './types';

async function jsonOrThrow<T>(r: Response): Promise<T> {
  if (!r.ok) {
    let msg = `${r.status} ${r.statusText}`;
    try { const b = await r.json(); if (b && b.error) msg = b.error; } catch { /* ignore */ }
    throw new Error(msg);
  }
  return r.json() as Promise<T>;
}

export async function getClusters(): Promise<Cluster[]> {
  const r = await fetch('/api/v1/clusters');
  const body = await jsonOrThrow<{ clusters: Cluster[] }>(r);
  return body.clusters ?? [];
}
export async function getConfig(): Promise<KapishConfig> {
  return jsonOrThrow<KapishConfig>(await fetch('/api/v1/config'));
}
export async function putConfig(cfg: KapishConfig): Promise<void> {
  const r = await fetch('/api/v1/config', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg) });
  await jsonOrThrow<{ status: string }>(r);
}
export async function getMgmts(): Promise<Mgmts> {
  return jsonOrThrow<Mgmts>(await fetch('/api/v1/mgmts'));
}
export async function putMgmtsCurrent(name: string): Promise<void> {
  const r = await fetch('/api/v1/mgmts/current', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) });
  await jsonOrThrow<{ status: string }>(r);
}
export async function createSession(namespace: string, cluster: string): Promise<CreateSessionResp> {
  const r = await fetch('/api/v1/sessions', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ namespace, cluster }) });
  return jsonOrThrow<CreateSessionResp>(r);
}
```

- [ ] **Step 3:** `npm run build` — compiles clean.
- [ ] **Step 4: Commit**
```bash
git add internal/web/frontend/src/api/types.ts internal/web/frontend/src/api/client.ts
git commit -m "feat(web): typed API client (clusters, config, mgmts, sessions)"
```

---

## Task 6: SSE stream subscriber

**Files:**
- Create: `internal/web/frontend/src/api/stream.ts`

- [ ] **Step 1:** `src/api/stream.ts`:
```ts
import type { Cluster } from './types';

export interface ClusterStreamHandlers {
  onSync: () => void; // re-fetch the snapshot
  onCluster: (type: 'added' | 'modified' | 'deleted', cluster: Cluster) => void;
  onError?: (e: Event) => void;
}

// subscribeClusterStream opens an EventSource on /api/v1/clusters/stream and
// dispatches to handlers. Returns a close() func. EventSource auto-reconnects
// on transport errors; on reconnect the server sends `event: sync` again, so
// onSync should re-fetch the snapshot.
export function subscribeClusterStream(h: ClusterStreamHandlers): () => void {
  const es = new EventSource('/api/v1/clusters/stream');
  es.addEventListener('sync', () => h.onSync());
  es.addEventListener('cluster', (ev) => {
    try {
      const m = JSON.parse((ev as MessageEvent).data) as { type: 'added' | 'modified' | 'deleted'; cluster: Cluster };
      h.onCluster(m.type, m.cluster);
    } catch { /* ignore malformed */ }
  });
  if (h.onError) es.onerror = h.onError;
  return () => es.close();
}
```

- [ ] **Step 2:** `npm run build` — compiles clean.
- [ ] **Step 3: Commit**
```bash
git add internal/web/frontend/src/api/stream.ts
git commit -m "feat(web): SSE cluster-stream subscriber"
```

---

## Task 7: WebSocket framing helper + TerminalSession component

**Files:**
- Create: `internal/web/frontend/src/api/ws.ts`, `internal/web/frontend/src/components/TerminalSession.tsx`

- [ ] **Step 1:** `src/api/ws.ts` — framing constants + helpers:
```ts
export const FRAME_DATA = 0x00;
export const FRAME_RESIZE = 0x01;
export const FRAME_PING = 0x02;

const enc = new TextEncoder();
const dec = new TextDecoder();

export function dataFrame(s: string): Uint8Array {
  const body = enc.encode(s);
  const out = new Uint8Array(body.length + 1);
  out[0] = FRAME_DATA;
  out.set(body, 1);
  return out;
}
export function resizeFrame(cols: number, rows: number): Uint8Array {
  const body = enc.encode(JSON.stringify({ cols, rows }));
  const out = new Uint8Array(body.length + 1);
  out[0] = FRAME_RESIZE;
  out.set(body, 1);
  return out;
}
export function pingFrame(): Uint8Array { return new Uint8Array([FRAME_PING]); }

// decodeIncoming returns the data payload string for a 0x00 frame, or null for
// pong / unknown frames.
export function decodeIncoming(buf: ArrayBuffer): string | null {
  const u = new Uint8Array(buf);
  if (u.length === 0) return null;
  if (u[0] === FRAME_DATA) return dec.decode(u.subarray(1));
  return null; // 0x02 pong etc.
}
```

- [ ] **Step 2:** `src/components/TerminalSession.tsx` — owns the xterm.js instance + WS lifecycle:
```tsx
import * as React from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { createSession } from '../api/client';
import { dataFrame, resizeFrame, pingFrame, decodeIncoming } from '../api/ws';
import { TerminalPanel } from '../ui/TerminalPanel';

export interface TerminalSessionProps {
  namespace: string;
  cluster: string;
  phase?: string;
  onDisconnect: () => void;
  onError?: (msg: string) => void;
}

export function TerminalSession({ namespace, cluster, phase, onDisconnect, onError }: TerminalSessionProps) {
  const hostRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    let disposed = false;
    const term = new Terminal({
      fontFamily: '"Geist Mono", ui-monospace, monospace',
      fontSize: 13,
      theme: { background: '#08090c' },
      cursorBlink: true,
      convertEol: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    fit.fit();

    let ws: WebSocket | null = null;
    let pingTimer: number | undefined;

    (async () => {
      try {
        const sess = await createSession(namespace, cluster);
        if (disposed) return;
        const wsURL = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + sess.wsUrl + '?token=' + encodeURIComponent(sess.wsToken);
        ws = new WebSocket(wsURL);
        ws.binaryType = 'arraybuffer';
        ws.onopen = () => {
          ws!.send(resizeFrame(term.cols, term.rows));
          pingTimer = window.setInterval(() => { try { ws?.send(pingFrame()); } catch { /* ignore */ } }, 30000);
        };
        ws.onmessage = (ev) => {
          const s = typeof ev.data === 'string' ? null : decodeIncoming(ev.data as ArrayBuffer);
          if (s) term.write(s);
        };
        ws.onclose = () => { if (!disposed) onDisconnect(); };
        ws.onerror = () => { if (!disposed && onError) onError('terminal connection error'); };
        term.onData((d) => { try { ws?.send(dataFrame(d)); } catch { /* ignore */ } });
      } catch (e) {
        if (!disposed && onError) onError(e instanceof Error ? e.message : String(e));
        if (!disposed) onDisconnect();
      }
    })();

    const onResize = () => {
      try {
        fit.fit();
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(resizeFrame(term.cols, term.rows));
      } catch { /* ignore */ }
    };
    window.addEventListener('resize', onResize);

    return () => {
      disposed = true;
      window.removeEventListener('resize', onResize);
      if (pingTimer) window.clearInterval(pingTimer);
      try { ws?.close(); } catch { /* ignore */ }
      term.dispose();
    };
  }, [namespace, cluster, onDisconnect, onError]);

  return <TerminalPanel cluster={cluster} namespace={namespace} phase={phase} terminalRef={hostRef} onDisconnect={onDisconnect} />;
}
```

- [ ] **Step 3:** `npm run build` — compiles clean (xterm + addon-fit imports resolve; `@xterm/xterm/css/xterm.css` was imported in `main.tsx`).
- [ ] **Step 4: Commit**
```bash
git add internal/web/frontend/src/api/ws.ts internal/web/frontend/src/components/TerminalSession.tsx
git commit -m "feat(web): WS framing helper + xterm.js TerminalSession"
```

---

## Task 8: MgmtPicker component

**Files:**
- Create: `internal/web/frontend/src/components/MgmtPicker.tsx`

- [ ] **Step 1:** `src/components/MgmtPicker.tsx` — a dropdown listing mgmt entries:
```tsx
import * as React from 'react';
import { getMgmts, putMgmtsCurrent } from '../api/client';
import type { Mgmts } from '../api/types';
import { IconCheck } from '../icons/Icon';

export function MgmtPicker({ onClose, onSwitched }: { onClose: () => void; onSwitched: (name: string) => void }) {
  const [data, setData] = React.useState<Mgmts | null>(null);
  const [err, setErr] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    getMgmts().then(setData).catch((e) => setErr(e instanceof Error ? e.message : String(e)));
  }, []);

  React.useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', k);
    return () => document.removeEventListener('keydown', k);
  }, [onClose]);

  const pick = async (name: string) => {
    if (busy || (data && name === data.current)) { onClose(); return; }
    setBusy(true);
    try {
      await putMgmtsCurrent(name);
      onSwitched(name);
      onClose();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  return (
    <div className="absolute left-4 top-12 z-30 w-72 rounded-md bg-bg-2 border border-border shadow-lg p-1">
      {err && <div className="px-3 py-2 text-xs text-error">{err}</div>}
      {!data && !err && <div className="px-3 py-2 text-xs text-muted">Loading…</div>}
      {data?.entries.map((e) => {
        const cur = e.name === data.current;
        return (
          <button key={e.name} onClick={() => pick(e.name)} disabled={busy}
            className="w-full flex items-center gap-2 px-3 py-1.5 rounded-sm text-sm text-left text-text-2 hover:bg-surface-2 hover:text-text disabled:opacity-50">
            <span className="w-4">{cur && <IconCheck size={14} className="text-success" />}</span>
            <span className="font-mono">{e.name}</span>
            {e.context && <span className="ml-auto text-2xs text-dim font-mono">{e.context}</span>}
          </button>
        );
      })}
    </div>
  );
}
```

- [ ] **Step 2:** `npm run build` — compiles clean.
- [ ] **Step 3: Commit**
```bash
git add internal/web/frontend/src/components/MgmtPicker.tsx
git commit -m "feat(web): MgmtPicker dropdown"
```

---

## Task 9: SettingsView — wired to GET/PUT /config

**Files:**
- Create: `internal/web/frontend/src/SettingsView.tsx`

- [ ] **Step 1:** `src/SettingsView.tsx` — header + tabs + the settings form, reading/writing real config:
```tsx
import * as React from 'react';
import { AppHeader } from './ui/AppHeader';
import { SettingsTabs, Tab } from './ui/SettingsTabs';
import { SettingsSectionForm, SettingsValue, Shell } from './ui/SettingsSectionForm';
import { Select } from './ui/Field';
import { Button } from './ui/Button';
import { getConfig, putConfig } from './api/client';
import type { KapishConfig } from './api/types';
import { KV } from './ui/KVList';

const TABS: Tab[] = [
  { id: 'shell', label: 'Shell & prompt' },
  { id: 'env', label: 'Env vars' },
  { id: 'aliases', label: 'Aliases' },
  { id: 'cwd', label: 'Working dir' },
  { id: 'theme', label: 'Theme' },
];

function kvFromRecord(r?: Record<string, string>): KV[] {
  return Object.entries(r ?? {}).map(([k, v]) => ({ k, v }));
}
function recordFromKV(items: KV[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const { k, v } of items) if (k) out[k] = v;
  return out;
}

export function SettingsView({ mgmtCluster, onClose }: { mgmtCluster: string; onClose: () => void }) {
  const [cfg, setCfg] = React.useState<KapishConfig | null>(null);
  const [tab, setTab] = React.useState('shell');
  const [val, setVal] = React.useState<SettingsValue | null>(null);
  const [err, setErr] = React.useState<string | null>(null);
  const [saving, setSaving] = React.useState(false);
  const [savedAt, setSavedAt] = React.useState<number | null>(null);

  React.useEffect(() => {
    getConfig().then((c) => {
      setCfg(c);
      const shell = (c.shell.command?.split('/').pop() as Shell) || 'zsh';
      setVal({
        shell: (['zsh', 'bash', 'fish'].includes(shell) ? shell : 'zsh') as Shell,
        prompt: c.shell.prompt ?? '[{cluster}] ',
        cwd: c.shell.cwd ?? '',
        env: kvFromRecord(c.shell.env),
        aliases: kvFromRecord(c.shell.aliases),
      });
    }).catch((e) => setErr(e instanceof Error ? e.message : String(e)));
  }, []);

  const save = async (nextCfg: KapishConfig) => {
    setSaving(true);
    try {
      await putConfig(nextCfg);
      setCfg(nextCfg);
      setSavedAt(Date.now());
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  const applyForm = (next: SettingsValue) => {
    setVal(next);
    if (!cfg) return;
    const merged: KapishConfig = {
      ...cfg,
      shell: {
        ...cfg.shell,
        command: next.shell === 'zsh' && !cfg.shell.command ? '' : cfg.shell.command, // keep command if set; shell select is best-effort
        cwd: next.cwd,
        prompt: next.prompt,
        env: recordFromKV(next.env),
        aliases: recordFromKV(next.aliases),
      },
    };
    save(merged);
  };

  const toggleTheme = () => {
    if (!cfg) return;
    const theme = cfg.ui.theme === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    save({ ...cfg, ui: { ...cfg.ui, theme } });
  };

  return (
    <div className="h-screen w-screen flex flex-col bg-bg text-text">
      <AppHeader mgmtCluster={mgmtCluster} onSettings={onClose} />
      <div className="flex-1 min-h-0 flex">
        <SettingsTabs tabs={TABS} active={tab} onSelect={setTab} />
        <main className="flex-1 min-w-0 overflow-y-auto p-6">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-semibold">{TABS.find((t) => t.id === tab)?.label}</h1>
            <div className="flex items-center gap-3">
              {saving && <span className="text-2xs text-muted">saving…</span>}
              {!saving && savedAt && <span className="text-2xs text-success">saved</span>}
              <Button variant="secondary" size="sm" onClick={onClose}>Close</Button>
            </div>
          </div>
          {err && <div className="mb-4 text-sm text-error">{err}</div>}
          {!val && !err && <div className="text-sm text-muted">Loading…</div>}
          {val && tab === 'theme' && cfg && (
            <Select label="Theme" value={cfg.ui.theme as 'dark' | 'light'} options={['dark', 'light']} onChange={toggleTheme as any} />
          )}
          {val && tab !== 'theme' && <SettingsSectionForm value={val} onChange={applyForm} />}
        </main>
      </div>
    </div>
  );
}
```

(Note: the `SettingsSectionForm` shows all of shell/prompt/cwd/env/aliases on one form; the tabs are mostly decorative here — that's acceptable for v1. The `theme` tab is the one special case.)

- [ ] **Step 2:** `npm run build` — compiles clean. (If `tsc` complains about the `onChange={toggleTheme as any}` cast or unused vars, tidy them — but keep the behavior.)
- [ ] **Step 3: Commit**
```bash
git add internal/web/frontend/src/SettingsView.tsx
git commit -m "feat(web): SettingsView wired to GET/PUT /api/v1/config"
```

---

## Task 10: App.tsx — assemble the main view, wire everything

**Files:**
- Modify: `internal/web/frontend/src/App.tsx`

- [ ] **Step 1:** Rewrite `src/App.tsx`:
```tsx
import * as React from 'react';
import { AppHeader } from './ui/AppHeader';
import { FilterInput } from './ui/FilterInput';
import { ClusterListRow } from './ui/ClusterListRow';
import { ClusterListSkeleton } from './ui/ClusterListSkeleton';
import { SelectClusterEmpty, NoClustersFoundEmpty } from './ui/EmptyState';
import { ConfirmDialog } from './ui/ConfirmDialog';
import { Toast, ToastStack } from './ui/Toast';
import { ErrorBanner } from './ui/ErrorBanner';
import { MgmtPicker } from './components/MgmtPicker';
import { TerminalSession } from './components/TerminalSession';
import { SettingsView } from './SettingsView';
import { getClusters, getMgmts } from './api/client';
import { subscribeClusterStream } from './api/stream';
import type { Cluster } from './api/types';

type View = 'main' | 'settings';

export default function App() {
  const [clusters, setClusters] = React.useState<Cluster[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [loadErr, setLoadErr] = React.useState<string | null>(null);
  const [filter, setFilter] = React.useState('');
  const [selectedKey, setSelectedKey] = React.useState<string | null>(null);
  const [pendingCluster, setPendingCluster] = React.useState<Cluster | null>(null); // confirm dialog target
  const [confirmFailed, setConfirmFailed] = React.useState<Cluster | null>(null);
  const [view, setView] = React.useState<View>('main');
  const [mgmtPickerOpen, setMgmtPickerOpen] = React.useState(false);
  const [mgmtCluster, setMgmtCluster] = React.useState('—');
  const [toasts, setToasts] = React.useState<{ id: number; tone: 'error' | 'info'; title: string }[]>([]);
  const toastId = React.useRef(0);
  const pushToast = (tone: 'error' | 'info', title: string) => {
    const id = ++toastId.current;
    setToasts((t) => [...t, { id, tone, title }]);
    window.setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 5000);
  };

  const refresh = React.useCallback(() => {
    setLoading(true);
    getClusters().then((cs) => { setClusters(cs); setLoadErr(null); }).catch((e) => setLoadErr(e instanceof Error ? e.message : String(e))).finally(() => setLoading(false));
  }, []);

  React.useEffect(() => { refresh(); getMgmts().then((m) => setMgmtCluster(m.current || '—')).catch(() => {}); }, [refresh]);

  React.useEffect(() => {
    const close = subscribeClusterStream({
      onSync: () => refresh(),
      onCluster: (type, c) => {
        setClusters((prev) => {
          const key = c.namespace + '/' + c.name;
          const without = prev.filter((p) => p.namespace + '/' + p.name !== key);
          if (type === 'deleted') return without;
          return [...without, c].sort((a, b) => (a.namespace !== b.namespace ? a.namespace.localeCompare(b.namespace) : a.name.localeCompare(b.name)));
        });
      },
    });
    return close;
  }, [refresh]);

  const sorted = React.useMemo(
    () => [...clusters].sort((a, b) => (a.namespace !== b.namespace ? a.namespace.localeCompare(b.namespace) : a.name.localeCompare(b.name))),
    [clusters],
  );
  const filtered = React.useMemo(
    () => (!filter ? sorted : sorted.filter((c) => c.name.includes(filter) || c.namespace.includes(filter))),
    [sorted, filter],
  );
  const keyOf = (c: Cluster) => c.namespace + '/' + c.name;
  const active = clusters.find((c) => keyOf(c) === selectedKey) ?? null;

  const tryConnect = (c: Cluster) => {
    if (c.phase === 'Failed' || c.phase === 'Deleting') { setConfirmFailed(c); return; }
    doConnect(c);
  };
  const doConnect = (c: Cluster) => {
    if (active && keyOf(active) !== keyOf(c)) { setPendingCluster(c); return; } // confirm replace
    setSelectedKey(keyOf(c));
  };

  if (view === 'settings') {
    return <SettingsView mgmtCluster={mgmtCluster} onClose={() => setView('main')} />;
  }

  return (
    <div className="h-screen w-screen flex flex-col bg-bg text-text font-sans relative">
      <AppHeader
        mgmtCluster={mgmtCluster}
        onPickMgmt={() => setMgmtPickerOpen((v) => !v)}
        onRefresh={refresh}
        refreshing={loading}
        onSettings={() => setView('settings')}
      />
      {mgmtPickerOpen && (
        <MgmtPicker onClose={() => setMgmtPickerOpen(false)} onSwitched={(name) => { setMgmtCluster(name); setSelectedKey(null); refresh(); }} />
      )}
      <div className="flex-1 min-h-0 flex">
        <aside className="flex-shrink-0 w-[340px] border-r border-border bg-bg-2 flex flex-col">
          <div className="p-3 border-b border-border">
            <FilterInput value={filter} onChange={setFilter} hint={`${filtered.length} matches`} />
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto">
            {loadErr && <ErrorBanner title="Couldn't load clusters" body={loadErr} onRetry={refresh} />}
            {loading && clusters.length === 0 && !loadErr && <ClusterListSkeleton />}
            {!loading && !loadErr && filtered.length === 0 && <NoClustersFoundEmpty onClear={() => setFilter('')} />}
            {filtered.map((c) => (
              <ClusterListRow key={keyOf(c)} name={c.name} namespace={c.namespace} phase={c.phase} version={c.version} provider={c.provider}
                selected={selectedKey === keyOf(c)} onClick={() => tryConnect(c)} onConnect={() => tryConnect(c)} />
            ))}
          </div>
        </aside>
        {active ? (
          <TerminalSession namespace={active.namespace} cluster={active.name} phase={active.phase}
            onDisconnect={() => setSelectedKey(null)} onError={(m) => pushToast('error', m)} />
        ) : (
          <div className="flex-1 flex"><SelectClusterEmpty /></div>
        )}
      </div>

      <ConfirmDialog open={!!confirmFailed} title={confirmFailed ? `${confirmFailed.name} is ${confirmFailed.phase}` : ''}
        body="Some kubectl calls may hang. Spawn a shell anyway?" confirmLabel="Continue anyway"
        onConfirm={() => { const c = confirmFailed!; setConfirmFailed(null); doConnect(c); }} onCancel={() => setConfirmFailed(null)} />

      <ConfirmDialog open={!!pendingCluster} title="Disconnect current shell?"
        body="The kubeconfig for the current session is discarded." confirmLabel="Disconnect" tone="danger"
        onConfirm={() => { const c = pendingCluster!; setPendingCluster(null); setSelectedKey(keyOf(c)); }} onCancel={() => setPendingCluster(null)} />

      <ToastStack>
        {toasts.map((t) => <Toast key={t.id} tone={t.tone} title={t.title} onClose={() => setToasts((x) => x.filter((y) => y.id !== t.id))} />)}
      </ToastStack>
    </div>
  );
}
```

- [ ] **Step 2:** `npm run build` — compiles clean (fix any `tsc` strict-mode complaints — unused vars, `as any` casts — without changing behavior).
- [ ] **Step 3: Commit**
```bash
git add internal/web/frontend/src/App.tsx
git commit -m "feat(web): App.tsx — cluster list + terminal + settings + mgmt picker wired to API"
```

---

## Task 11: `--dev` mode — Go server proxies to Vite

**Files:**
- Modify: `cmd/kapish/serve.go`
- Modify: `internal/web/server.go`

- [ ] **Step 1:** Add a `Dev bool` and `DevTarget string` to `web.Options`. When `Dev` is true, instead of `http.FileServer(http.FS(frontendRoot()))` for `/`, register a reverse proxy (`net/http/httputil.NewSingleHostReverseProxy`) to `DevTarget` (the Vite dev server URL, e.g. `http://127.0.0.1:5173`). The `/api/...` routes still take precedence (more specific). Add `internal/web/dev.go` with the proxy setup; `routes()` branches on `s.opts.Dev`.

- [ ] **Step 2:** In `cmd/kapish/serve.go` `runServe`: if `--dev` is set, (a) the server's `Dev=true, DevTarget="http://127.0.0.1:5173"` (Vite's default port — or read `KAPISH_VITE_URL` env if you want it configurable); (b) print a note: `dev mode: run 'npm run dev' in internal/web/frontend (proxying / to http://127.0.0.1:5173)`; (c) DON'T open the browser to the Go server — print the Vite URL instead, since that's where HMR lives. Actually simpler: in dev mode just don't auto-open at all and print both URLs (Go API at <addr>, Vite at :5173).

- [ ] **Step 3:** `go build ./... && go test ./... -count=1 && go vet ./...` — green. `go build ./cmd/kapish && ./bin/kapish serve --help` — `--dev` still listed (it was added in Plan 4).

- [ ] **Step 4: Commit**
```bash
git add cmd/kapish/serve.go internal/web/server.go internal/web/dev.go
git commit -m "feat(web): --dev mode proxies / to the Vite dev server"
```

---

## Task 12: Production build + commit dist/

**Files:**
- Modify: `internal/web/frontend/dist/*` (Vite build output — overwrites the placeholder)

- [ ] **Step 1:** Build the frontend:
```sh
cd internal/web/frontend && npm install && npm run build
```
Expected: `dist/index.html` + `dist/assets/index-<hash>.js` + `dist/assets/index-<hash>.css` produced. Verify `dist/index.html` references the hashed assets and has `<div id="root">`.

- [ ] **Step 2:** From repo root, verify the embed picks up the real build:
```sh
go build ./...
go test ./internal/web -run TestServesIndexAtRoot -v
```
The `TestServesIndexAtRoot` test asserts the `/` response contains "kapish" — the Vite build's `index.html` has `<title>kapish</title>`, so it still passes. (If the test asserted something more specific that the new build doesn't have, update the test to assert a stable string like `<div id="root">` instead.)

- [ ] **Step 3:** Manual smoke (optional but recommended): run `./bin/kapish serve --port 8765 --no-open --kubeconfig <a real or fake kubeconfig>` and `curl -s http://127.0.0.1:8765/ | head -5` — should be the Vite-built HTML, not the Plan 4 placeholder. `curl -s http://127.0.0.1:8765/api/v1/health` — `{"status":"ok"}`. (With a fake kubeconfig the server still starts; the cluster list will be empty/erroring but the static UI serves fine.)

- [ ] **Step 4:** Commit the build output:
```bash
git add internal/web/frontend/dist
git commit -m "build(web): production frontend build → dist/"
```

> **Note for maintainers (add to README in Task 13):** the committed `dist/` is the source of truth for `//go:embed`. After any change to `internal/web/frontend/src/`, run `make frontend` and commit the regenerated `dist/`. (A pre-commit hook or CI check could enforce this later — out of scope here.)

---

## Task 13: Full verification + README update + final review

**Files:**
- Modify: `README.md`

- [ ] **Step 1:** `make test` — all 8 packages green.
- [ ] **Step 2:** `go vet ./... && go mod tidy` — clean, no diff.
- [ ] **Step 3:** `make build && ./bin/kapish --help` — `serve` and `version` and `config` all listed. `./bin/kapish serve --help` — `--port`, `--bind`, `--no-open`, `--dev`.
- [ ] **Step 4:** `go install ./cmd/kapish && "$(go env GOPATH)/bin/kapish" version` — works (and crucially, `go install` succeeded *with* the embedded `dist/`, proving the committed build output makes `go install` self-contained).
- [ ] **Step 5:** Update `README.md`: the `kapish serve` line now actually serves a UI; add a "Web UI development" section: `make frontend` to rebuild + commit `dist/`, `kapish serve --dev` + `npm run dev` for HMR. Remove the "(Plan 4 + Plan 5)" parenthetical from the usage block.
- [ ] **Step 6:** Commit README:
```bash
git add README.md
git commit -m "docs: README — web UI usage + frontend dev workflow"
```
- [ ] **Step 7:** If `go mod tidy` changed anything, commit it (`chore: tidy go.mod after Plan 5`).

---

## Plan 5 exit criteria

- [ ] `make test` green across all packages.
- [ ] `make frontend` produces `internal/web/frontend/dist/` and the result is committed.
- [ ] `make build && ./bin/kapish serve --port N --no-open` serves the React UI at `/` (verified via `curl` — Vite-built HTML, not the placeholder); `/api/v1/health` still returns `{"status":"ok"}`.
- [ ] The UI (when opened in a browser against a real CAPI mgmt cluster): cluster list renders + updates live via SSE; clicking a cluster opens an in-browser xterm.js terminal connected via the WebSocket-PTY bridge; the settings page reads and writes config; the mgmt chip dropdown switches management clusters. (This last item is browser-only verification — note it explicitly as "manually verified" or "not verified, needs a live cluster" in the final report.)
- [ ] `kapish serve --dev` proxies `/` to `http://127.0.0.1:5173`; `npm run dev` in `internal/web/frontend` proxies `/api` (incl. WS) back to the Go server.
- [ ] `go install github.com/v4run/kapish/cmd/kapish@latest` ships the UI (the committed `dist/` is embedded; no Node toolchain needed at install time).
- [ ] `go vet ./...` clean; `go mod tidy` no-op.

When all boxes are checked, Plan 5 — and the kapish project's initial implementation — is done. The full TUI + Web UI tool is shippable.
