# kapish · Web UI handoff

> Source: the Claude Design handoff provided by the user. This is the authoritative reference for the Web UI's visual layer — design tokens, brand assets, icons, and React component implementations. Plan 5 tasks drop these files into `internal/web/frontend/src/` (mapping below). Where the handoff implies features beyond v1 scope (Hooks/Keybinds settings tabs, `nu`/`pwsh` shell options, per-cluster/per-namespace scope subtitle), v1 omits them.

React 18 + TypeScript + Tailwind + xterm.js. **No external UI library** (no shadcn / MUI / Radix). Inline SVG only. The kapish binary embeds these assets, so the bundle stays small and predictable. Components avoid exotic React APIs — Solid swap later should be mechanical.

---

## 1) Design tokens

### `tailwind.config.js` — `theme.extend`

```js
// tailwind.config.js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{ts,tsx,html}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        bg:        'rgb(var(--bg) / <alpha-value>)',
        'bg-2':    'rgb(var(--bg-2) / <alpha-value>)',
        surface:   'rgb(var(--surface) / <alpha-value>)',
        'surface-2':'rgb(var(--surface-2) / <alpha-value>)',
        'surface-3':'rgb(var(--surface-3) / <alpha-value>)',
        border:    'rgb(var(--border) / <alpha-value>)',
        'border-2':'rgb(var(--border-2) / <alpha-value>)',
        text:      'rgb(var(--text) / <alpha-value>)',
        'text-2':  'rgb(var(--text-2) / <alpha-value>)',
        muted:     'rgb(var(--muted) / <alpha-value>)',
        dim:       'rgb(var(--dim) / <alpha-value>)',
        primary:   'rgb(var(--primary) / <alpha-value>)',
        accent:    'rgb(var(--accent) / <alpha-value>)',
        success:   'rgb(var(--success) / <alpha-value>)',
        warning:   'rgb(var(--warning) / <alpha-value>)',
        error:     'rgb(var(--error) / <alpha-value>)',
        info:      'rgb(var(--info) / <alpha-value>)',
        'p-aws':     'rgb(var(--p-aws) / <alpha-value>)',
        'p-gcp':     'rgb(var(--p-gcp) / <alpha-value>)',
        'p-azure':   'rgb(var(--p-azure) / <alpha-value>)',
        'p-vsphere': 'rgb(var(--p-vsphere) / <alpha-value>)',
        'p-hetzner': 'rgb(var(--p-hetzner) / <alpha-value>)',
      },
      fontFamily: {
        sans: ['Geist', 'ui-sans-serif', 'system-ui', '-apple-system', 'Segoe UI', 'sans-serif'],
        mono: ['"Geist Mono"', 'ui-monospace', '"JetBrains Mono"', '"SF Mono"', 'Menlo', 'Consolas', 'monospace'],
      },
      fontSize: {
        '2xs': ['10px', { lineHeight: '1.4' }],
        xs:    ['11px', { lineHeight: '1.45' }],
        sm:    ['13px', { lineHeight: '1.5' }],
        base:  ['14px', { lineHeight: '1.5' }],
        md:    ['15px', { lineHeight: '1.5' }],
        lg:    ['17px', { lineHeight: '1.45' }],
        xl:    ['20px', { lineHeight: '1.35' }],
        '2xl': ['26px', { lineHeight: '1.25', letterSpacing: '-0.01em' }],
        '3xl': ['34px', { lineHeight: '1.15', letterSpacing: '-0.02em' }],
        '4xl': ['52px', { lineHeight: '1.05', letterSpacing: '-0.025em' }],
      },
      fontWeight: { normal: '400', medium: '500', semibold: '600', bold: '700' },
      spacing: {
        0.5: '2px', 1: '4px', 1.5: '6px', 2: '8px', 2.5: '10px',
        3: '12px', 4: '16px', 5: '20px', 6: '24px', 7: '28px',
        8: '32px', 10: '40px', 12: '48px', 14: '56px', 16: '64px',
      },
      borderRadius: { none: '0', xs: '3px', sm: '4px', DEFAULT: '6px', md: '8px', lg: '10px', xl: '12px', '2xl': '16px', full: '9999px' },
      boxShadow: {
        xs: '0 1px 1px rgb(0 0 0 / .25)',
        sm: '0 1px 2px rgb(0 0 0 / .35)',
        md: '0 4px 16px rgb(0 0 0 / .35), 0 1px 2px rgb(0 0 0 / .35)',
        lg: '0 16px 48px rgb(0 0 0 / .45)',
        xl: '0 24px 80px rgb(0 0 0 / .45), 0 0 0 1px rgb(255 255 255 / .04)',
        focus: '0 0 0 2px rgb(var(--primary) / .35)',
        'focus-violet': '0 0 0 2px rgb(var(--accent) / .35)',
      },
    },
  },
};
```

### CSS custom properties — `src/styles/tokens.css`

```css
:root, :root[data-theme="dark"] {
  --bg: 10 11 14; --bg-2: 15 17 22; --surface: 20 22 28; --surface-2: 27 30 38; --surface-3: 35 39 49;
  --border: 38 42 52; --border-2: 53 58 71; --text: 232 233 238; --text-2: 184 186 196; --muted: 122 126 138; --dim: 79 83 94;
  --primary: 141 213 233; --accent: 180 141 255; --success: 149 220 161; --warning: 240 196 116; --error: 232 122 98; --info: 141 187 233;
  --p-aws: 233 178 90; --p-gcp: 141 173 233; --p-azure: 141 200 233; --p-vsphere: 141 220 162; --p-hetzner: 232 122 98;
}
:root[data-theme="light"] {
  --bg: 240 238 233; --bg-2: 247 245 240; --surface: 255 255 255; --surface-2: 248 246 241; --surface-3: 240 237 230;
  --border: 220 215 205; --border-2: 198 192 180; --text: 20 22 28; --text-2: 60 64 74; --muted: 108 112 124; --dim: 150 154 165;
  --primary: 20 130 160; --accent: 109 60 232; --success: 36 144 74; --warning: 168 105 16; --error: 193 61 41; --info: 40 110 196;
  --p-aws: 192 122 20; --p-gcp: 40 110 196; --p-azure: 20 130 178; --p-vsphere: 36 144 74; --p-hetzner: 193 61 41;
}
html { font-family: var(--font-sans, Geist, system-ui, sans-serif); background: rgb(var(--bg)); color: rgb(var(--text)); font-feature-settings: "ss01","cv01","cv11"; }
code, pre, .mono { font-family: 'Geist Mono', ui-monospace, monospace; }
@keyframes blink { 0%,50%{opacity:1} 50.01%,100%{opacity:0} }
.cursor { animation: blink 1s steps(2) infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1.1s linear infinite; }
```

---

## 2) Brand assets — inline SVG

### `src/brand/KapishMark.tsx`

```tsx
import * as React from 'react';
type Props = { size?: number; className?: string; accent?: string; violet?: string; mono?: boolean };
export function KapishMark({ size = 32, className, accent = 'currentColor', violet, mono = false }: Props) {
  const A = accent;
  const V = mono ? accent : (violet ?? 'rgb(180 141 255)');
  const stroke = 2.6;
  const dot = 2.8;
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" className={className} aria-label="kapish">
      <line x1="9" y1="6" x2="9" y2="26" stroke={A} strokeWidth={stroke} strokeLinecap="round"/>
      <line x1="9" y1="16" x2="20" y2="7" stroke={V} strokeWidth={stroke} strokeLinecap="round"/>
      <line x1="9" y1="16" x2="24" y2="26" stroke={V} strokeWidth={stroke} strokeLinecap="round"/>
      <circle cx="9" cy="6" r={dot} fill={A}/>
      <circle cx="9" cy="26" r={dot} fill={A}/>
      <circle cx="9" cy="16" r={dot * 0.85} fill={A}/>
      <circle cx="20" cy="7" r={dot} fill={V}/>
      <circle cx="24" cy="26" r={dot} fill={V}/>
    </svg>
  );
}
```

### `src/brand/KapishLockup.tsx`

```tsx
import * as React from 'react';
import { KapishMark } from './KapishMark';
export function KapishWordmark({ className = '', cursor = true }: { className?: string; cursor?: boolean }) {
  return (
    <span className={`inline-flex items-baseline font-bold leading-none ${className}`}>
      <span className="font-sans tracking-tight">kapi</span>
      <span className="font-mono text-primary ml-[0.04em]">sh</span>
      {cursor && (<span aria-hidden className="cursor inline-block bg-primary ml-[0.10em]" style={{ width: '0.42em', height: '0.78em', transform: 'translateY(0.04em)', borderRadius: 1 }}/>)}
    </span>
  );
}
export function KapishLockup({ size = 28 }: { size?: number }) {
  return (
    <div className="inline-flex items-center gap-3">
      <KapishMark size={Math.round(size * 1.05)} />
      <KapishWordmark className="text-text" />
    </div>
  );
}
```

### Icon set — `src/icons/Icon.tsx`

```tsx
import * as React from 'react';
const I = ({ size = 16, className = '', children, label }: { size?: number; className?: string; children: React.ReactNode; label: string }) => (
  <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" className={className} role="img" aria-label={label}>{children}</svg>
);
export const IconSearch  = (p: any) => <I {...p} label="search"><circle cx="7" cy="7" r="4.5"/><path d="M11 11l3 3"/></I>;
export const IconClose   = (p: any) => <I {...p} label="close"><path d="M3.5 3.5l9 9M12.5 3.5l-9 9"/></I>;
export const IconRefresh = (p: any) => <I {...p} label="refresh"><path d="M13.5 3.5v3h-3"/><path d="M13 6.5A5 5 0 1 0 13.5 11"/></I>;
export const IconSettings= (p: any) => <I {...p} label="settings"><circle cx="8" cy="8" r="2"/><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.5 3.5l1.4 1.4M11.1 11.1l1.4 1.4M3.5 12.5l1.4-1.4M11.1 4.9l1.4-1.4"/></I>;
export const IconChevron = (p: any) => <I {...p} label="chevron"><path d="M6 4l4 4-4 4"/></I>;
export const IconPower   = (p: any) => <I {...p} label="disconnect"><path d="M5 4a4.5 4.5 0 1 0 6 0"/><path d="M8 1.5v6"/></I>;
export const IconCheck   = (p: any) => <I {...p} label="check"><path d="M3 8.5l3 3 7-7"/></I>;
export const IconWarn    = (p: any) => <I {...p} label="warning"><path d="M8 3l6 10H2z"/><path d="M8 7v3M8 11.5v.01"/></I>;
export const IconInfo    = (p: any) => <I {...p} label="info"><circle cx="8" cy="8" r="6.5"/><path d="M8 7.5v3.5M8 5v.01"/></I>;
export const IconError   = (p: any) => <I {...p} label="error"><circle cx="8" cy="8" r="6.5"/><path d="M5.5 5.5l5 5M10.5 5.5l-5 5"/></I>;
export const IconPlus    = (p: any) => <I {...p} label="add"><path d="M8 3v10M3 8h10"/></I>;
export const IconTrash   = (p: any) => <I {...p} label="remove"><path d="M3 4.5h10M6 4.5V3h4v1.5M5 4.5v8a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-8"/></I>;
export const IconTerminal= (p: any) => <I {...p} label="terminal"><rect x="1.5" y="2.5" width="13" height="11" rx="1.5"/><path d="M4 6l2 2-2 2M8 10h4"/></I>;
```

---

## 3) Components

### `src/ui/Button.tsx`

```tsx
import * as React from 'react';
type Variant = 'primary' | 'secondary' | 'icon' | 'danger';
type Size = 'sm' | 'md';
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant; size?: Size; leading?: React.ReactNode; trailing?: React.ReactNode;
}
const base = 'inline-flex items-center justify-center gap-2 font-medium rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus-visible:shadow-focus';
const sizes: Record<Size, string> = { sm: 'h-7 px-2.5 text-xs', md: 'h-9 px-3.5 text-sm' };
const variants: Record<Variant, string> = {
  primary: 'bg-primary text-bg hover:brightness-110 active:brightness-95',
  secondary: 'bg-surface text-text-2 border border-border hover:bg-surface-2 hover:text-text',
  icon: 'bg-transparent text-text-2 hover:bg-surface-2 hover:text-text aspect-square px-0',
  danger: 'bg-transparent text-error border border-error/50 hover:bg-error/10',
};
export function Button({ variant = 'secondary', size = 'md', leading, trailing, children, className = '', ...rest }: ButtonProps) {
  return (<button {...rest} className={`${base} ${sizes[size]} ${variants[variant]} ${className}`}>{leading}{children}{trailing}</button>);
}
```

### `src/ui/AppHeader.tsx`

```tsx
import * as React from 'react';
import { KapishLockup } from '../brand/KapishLockup';
import { Button } from './Button';
import { IconRefresh, IconSettings, IconChevron } from '../icons/Icon';
export interface AppHeaderProps {
  mgmtCluster: string; onPickMgmt?: () => void; onRefresh?: () => void; onSettings?: () => void; refreshing?: boolean; version?: string;
}
export function AppHeader({ mgmtCluster, onPickMgmt, onRefresh, onSettings, refreshing, version }: AppHeaderProps) {
  return (
    <header className="h-12 flex-shrink-0 flex items-center gap-4 px-4 border-b border-border bg-bg-2">
      <KapishLockup size={22} />
      {version && <span className="text-dim font-mono text-2xs">{version}</span>}
      <button onClick={onPickMgmt} className="ml-3 inline-flex items-center gap-2 px-2.5 h-7 rounded-md bg-surface border border-border text-text-2 hover:bg-surface-2 hover:text-text font-mono text-xs">
        <span className="size-1.5 rounded-full bg-success" />
        <span className="text-muted">mgmt</span>
        <span>{mgmtCluster}</span>
        <IconChevron size={12} className="text-muted" />
      </button>
      <div className="flex-1" />
      <Button variant="icon" size="sm" onClick={onRefresh} aria-label="Refresh"><IconRefresh size={14} className={refreshing ? 'spin' : ''} /></Button>
      <Button variant="icon" size="sm" onClick={onSettings} aria-label="Settings"><IconSettings size={14} /></Button>
    </header>
  );
}
```

### `src/ui/FilterInput.tsx`

```tsx
import * as React from 'react';
import { IconSearch, IconClose } from '../icons/Icon';
export interface FilterInputProps { value: string; onChange: (v: string) => void; placeholder?: string; hint?: React.ReactNode; autoFocus?: boolean; }
export function FilterInput({ value, onChange, placeholder = 'Search clusters…', hint, autoFocus }: FilterInputProps) {
  return (
    <div className="flex items-center gap-2 h-9 px-3 rounded-md bg-surface border border-border focus-within:border-primary/60 focus-within:shadow-focus transition-colors">
      <IconSearch size={14} className="text-muted shrink-0" />
      <input autoFocus={autoFocus} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} className="flex-1 bg-transparent outline-none text-sm text-text placeholder:text-muted" />
      {hint && <span className="text-2xs text-dim font-mono">{hint}</span>}
      {value && (<button onClick={() => onChange('')} className="text-muted hover:text-text" aria-label="Clear"><IconClose size={12} /></button>)}
    </div>
  );
}
```

### `src/ui/PhaseChip.tsx`

```tsx
import * as React from 'react';
export type Phase = 'Provisioned' | 'Provisioning' | 'Pending' | 'Failed' | 'Deleting';
const PHASE: Record<Phase, { color: string; bg: string; label: string }> = {
  Provisioned:  { color: 'text-success', bg: 'bg-success/15',  label: 'Provisioned' },
  Provisioning: { color: 'text-warning', bg: 'bg-warning/15',  label: 'Provisioning' },
  Pending:      { color: 'text-info',    bg: 'bg-info/15',     label: 'Pending' },
  Failed:       { color: 'text-error',   bg: 'bg-error/15',    label: 'Failed' },
  Deleting:     { color: 'text-muted',   bg: 'bg-surface-2',   label: 'Deleting' },
};
// Tolerate unknown/empty phase strings by falling back to a neutral chip.
export function PhaseChip({ phase }: { phase: string }) {
  const p = PHASE[phase as Phase] ?? { color: 'text-muted', bg: 'bg-surface-2', label: phase || 'Unknown' };
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 h-5 rounded-sm font-mono text-2xs font-medium ${p.color} ${p.bg}`}>
      <span className="size-1.5 rounded-full bg-current" />{p.label}
    </span>
  );
}
```

### `src/ui/ClusterListRow.tsx`

```tsx
import * as React from 'react';
import { PhaseChip } from './PhaseChip';
const PROVIDER_TINT: Record<string, string> = {
  aws: 'text-p-aws bg-p-aws/15', gcp: 'text-p-gcp bg-p-gcp/15', azure: 'text-p-azure bg-p-azure/15',
  vsphere: 'text-p-vsphere bg-p-vsphere/15', hetzner: 'text-p-hetzner bg-p-hetzner/15',
};
export interface ClusterListRowProps {
  name: string; namespace: string; phase: string; version: string; provider: string;
  selected?: boolean; onClick?: () => void; onConnect?: () => void;
}
export function ClusterListRow({ name, namespace, phase, version, provider, selected, onClick, onConnect }: ClusterListRowProps) {
  const tint = PROVIDER_TINT[provider] ?? 'text-muted bg-surface-2';
  return (
    <button onClick={onClick} onDoubleClick={onConnect}
      className={`w-full text-left grid items-center gap-3 grid-cols-[1fr_auto_auto_auto] px-3 py-2 border-l-2 ${selected ? 'bg-accent/12 border-accent text-text' : 'border-transparent text-text-2 hover:bg-surface-2'} transition-colors`}>
      <div className="min-w-0">
        <div className="font-mono text-sm font-medium text-text truncate">{name}</div>
        <div className="text-2xs text-muted font-mono truncate">{namespace}</div>
      </div>
      <span className={`inline-flex items-center gap-1.5 px-2 h-5 rounded-sm font-mono text-2xs ${tint}`}><span className="size-1.5 rounded-full bg-current" />{provider || '-'}</span>
      <span className="font-mono text-2xs text-muted">{version || '-'}</span>
      <PhaseChip phase={phase} />
    </button>
  );
}
```

### `src/ui/TerminalPanel.tsx`

```tsx
import * as React from 'react';
import { Button } from './Button';
import { IconPower, IconTerminal } from '../icons/Icon';
import { PhaseChip } from './PhaseChip';
export interface TerminalPanelProps {
  cluster: string; namespace?: string; phase?: string;
  terminalRef: React.Ref<HTMLDivElement>; onDisconnect?: () => void; toolbarExtra?: React.ReactNode;
}
export function TerminalPanel({ cluster, namespace, phase, terminalRef, onDisconnect, toolbarExtra }: TerminalPanelProps) {
  return (
    <section className="flex-1 min-w-0 flex flex-col bg-bg">
      <div className="h-10 flex-shrink-0 flex items-center gap-3 px-4 border-b border-border bg-bg-2">
        <IconTerminal size={14} className="text-muted" />
        <span className="font-mono text-sm font-medium text-text truncate">{cluster}</span>
        {namespace && <span className="font-mono text-xs text-muted">· {namespace}</span>}
        {phase && <PhaseChip phase={phase} />}
        <div className="flex-1" />
        {toolbarExtra}
        <Button variant="icon" size="sm" onClick={onDisconnect} aria-label="Disconnect" className="hover:text-error"><IconPower size={14} /></Button>
      </div>
      <div ref={terminalRef} className="flex-1 min-h-0 bg-[#08090c] font-mono" data-terminal />
    </section>
  );
}
```

### `src/ui/EmptyState.tsx`

```tsx
import * as React from 'react';
export interface EmptyStateProps { icon?: React.ReactNode; title: string; body?: string; action?: React.ReactNode; }
export function EmptyState({ icon, title, body, action }: EmptyStateProps) {
  return (
    <div className="h-full flex flex-col items-center justify-center px-8 text-center">
      {icon && <div className="text-muted mb-3">{icon}</div>}
      <div className="text-text text-md font-medium">{title}</div>
      {body && <p className="mt-1 text-sm text-text-2 max-w-xs">{body}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}
export const SelectClusterEmpty = () => (
  <EmptyState
    icon={<svg width={28} height={28} viewBox="0 0 32 32" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"><line x1="9" y1="6" x2="9" y2="26"/><line x1="9" y1="16" x2="20" y2="7"/><line x1="9" y1="16" x2="24" y2="26"/></svg>}
    title="Select a cluster to start a shell"
    body="Pick any cluster from the list. kapish will fetch its kubeconfig and spawn your configured shell." />
);
export const NoClustersFoundEmpty = ({ onClear }: { onClear?: () => void }) => (
  <EmptyState title="No clusters found" body="No clusters match your filters. Try clearing the search or switching management contexts."
    action={onClear && <button onClick={onClear} className="text-primary text-sm hover:underline">Clear filters</button>} />
);
```

### `src/ui/ConfirmDialog.tsx`

```tsx
import * as React from 'react';
import { Button } from './Button';
export interface ConfirmDialogProps {
  open: boolean; title: string; body?: React.ReactNode; confirmLabel?: string; cancelLabel?: string;
  tone?: 'default' | 'danger'; onConfirm: () => void; onCancel: () => void;
}
export function ConfirmDialog({ open, title, body, confirmLabel = 'Confirm', cancelLabel = 'Cancel', tone = 'default', onConfirm, onCancel }: ConfirmDialogProps) {
  React.useEffect(() => {
    if (!open) return;
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape') onCancel(); if (e.key === 'Enter') onConfirm(); };
    document.addEventListener('keydown', k);
    return () => document.removeEventListener('keydown', k);
  }, [open, onCancel, onConfirm]);
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/55 backdrop-blur-sm" onClick={onCancel}>
      <div role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()} className="w-[420px] rounded-xl bg-bg-2 border border-border shadow-xl p-5">
        <div className="text-md font-semibold text-text">{title}</div>
        {body && <div className="mt-2 text-sm text-text-2">{body}</div>}
        <div className="mt-5 flex gap-2 justify-end">
          <Button variant="secondary" onClick={onCancel}>{cancelLabel}</Button>
          <Button variant={tone === 'danger' ? 'danger' : 'primary'} onClick={onConfirm}>{confirmLabel}</Button>
        </div>
      </div>
    </div>
  );
}
```

### `src/ui/Toast.tsx`

```tsx
import * as React from 'react';
import { IconCheck, IconWarn, IconInfo, IconError, IconClose } from '../icons/Icon';
type Tone = 'success' | 'warning' | 'error' | 'info';
const ICON: Record<Tone, React.FC<any>> = { success: IconCheck, warning: IconWarn, error: IconError, info: IconInfo };
const TONE: Record<Tone, string> = {
  success: 'text-success border-success/30', warning: 'text-warning border-warning/30', error: 'text-error border-error/40', info: 'text-info border-info/30',
};
export interface ToastProps { tone?: Tone; title: string; body?: React.ReactNode; onClose?: () => void; }
export function Toast({ tone = 'info', title, body, onClose }: ToastProps) {
  const Icon = ICON[tone];
  return (
    <div role="status" className={`flex items-start gap-3 w-[360px] p-3 pr-2 rounded-lg bg-surface border ${TONE[tone]} shadow-md`}>
      <Icon size={16} className="mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-text">{title}</div>
        {body && <div className="text-xs text-text-2 mt-0.5">{body}</div>}
      </div>
      {onClose && (<button onClick={onClose} className="text-muted hover:text-text"><IconClose size={12} /></button>)}
    </div>
  );
}
export function ToastStack({ children }: { children: React.ReactNode }) {
  return <div className="fixed bottom-4 right-4 z-40 flex flex-col gap-2">{children}</div>;
}
```

### `src/ui/SettingsTabs.tsx`

```tsx
import * as React from 'react';
export interface Tab { id: string; label: string }
export function SettingsTabs({ tabs, active, onSelect }: { tabs: Tab[]; active: string; onSelect: (id: string) => void }) {
  return (
    <nav role="tablist" className="flex flex-col w-56 p-2 border-r border-border bg-bg-2">
      {tabs.map((t) => {
        const sel = t.id === active;
        return (
          <button key={t.id} role="tab" aria-selected={sel} onClick={() => onSelect(t.id)}
            className={`px-3 py-1.5 rounded-md text-sm text-left transition-colors mb-px ${sel ? 'bg-surface-2 text-text' : 'text-text-2 hover:bg-surface-2 hover:text-text'}`}>{t.label}</button>
        );
      })}
    </nav>
  );
}
```

### `src/ui/Field.tsx`

```tsx
import * as React from 'react';
export function TextField({ label, hint, value, onChange, placeholder, mono = false }: { label: string; hint?: string; value: string; onChange: (v: string) => void; placeholder?: string; mono?: boolean; }) {
  return (
    <label className="block">
      <div className="text-2xs uppercase tracking-wider text-muted mb-1.5">{label}</div>
      <input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className={`w-full h-9 px-3 rounded-md bg-bg border border-border focus:border-primary/60 focus:shadow-focus outline-none text-sm text-text placeholder:text-muted ${mono ? 'font-mono' : ''}`}/>
      {hint && <div className="mt-1.5 text-2xs text-dim">{hint}</div>}
    </label>
  );
}
export function Select<T extends string>({ label, value, options, onChange }: { label: string; value: T; options: T[]; onChange: (v: T) => void; }) {
  return (
    <label className="block">
      <div className="text-2xs uppercase tracking-wider text-muted mb-1.5">{label}</div>
      <div className="flex gap-1.5 flex-wrap">
        {options.map((o) => {
          const sel = o === value;
          return (<button key={o} type="button" onClick={() => onChange(o)} className={`h-8 px-3 rounded-md text-sm font-mono border ${sel ? 'bg-primary/15 text-primary border-primary' : 'bg-surface text-text-2 border-border hover:bg-surface-2'}`}>{o}</button>);
        })}
      </div>
    </label>
  );
}
```

### `src/ui/KVList.tsx`

```tsx
import * as React from 'react';
import { IconPlus, IconTrash } from '../icons/Icon';
export interface KV { k: string; v: string }
export function KVList({ label, hint, items, onChange, keyPlaceholder = 'KEY', valuePlaceholder = 'value' }: {
  label: string; hint?: string; items: KV[]; onChange: (next: KV[]) => void; keyPlaceholder?: string; valuePlaceholder?: string;
}) {
  const patch = (i: number, p: Partial<KV>) => onChange(items.map((it, idx) => idx === i ? { ...it, ...p } : it));
  const remove = (i: number) => onChange(items.filter((_, idx) => idx !== i));
  const add = () => onChange([...items, { k: '', v: '' }]);
  return (
    <div>
      <div className="flex items-baseline justify-between mb-1.5">
        <div className="text-2xs uppercase tracking-wider text-muted">{label}</div>
        {hint && <div className="text-2xs text-dim">{hint}</div>}
      </div>
      <div className="rounded-md border border-border divide-y divide-border bg-surface">
        {items.map((it, i) => (
          <div key={i} className="grid grid-cols-[150px_1fr_auto] gap-0 items-stretch">
            <input value={it.k} onChange={(e) => patch(i, { k: e.target.value })} placeholder={keyPlaceholder} className="h-8 px-2.5 bg-transparent font-mono text-xs text-primary placeholder:text-muted outline-none border-r border-border focus:bg-bg-2"/>
            <input value={it.v} onChange={(e) => patch(i, { v: e.target.value })} placeholder={valuePlaceholder} className="h-8 px-2.5 bg-transparent font-mono text-xs text-text-2 placeholder:text-muted outline-none focus:bg-bg-2"/>
            <button onClick={() => remove(i)} className="px-2 text-muted hover:text-error" aria-label="Remove"><IconTrash size={12}/></button>
          </div>
        ))}
        <button onClick={add} className="w-full h-8 flex items-center justify-center gap-1.5 text-xs text-muted hover:text-text hover:bg-surface-2"><IconPlus size={12}/> Add</button>
      </div>
    </div>
  );
}
```

### `src/ui/SettingsSectionForm.tsx`

```tsx
import * as React from 'react';
import { TextField, Select } from './Field';
import { KVList, KV } from './KVList';
// v1 supported shells only (handoff also listed nu/pwsh; deferred).
export type Shell = 'zsh' | 'bash' | 'fish';
export interface SettingsValue { shell: Shell; prompt: string; cwd: string; env: KV[]; aliases: KV[]; }
export function SettingsSectionForm({ value, onChange }: { value: SettingsValue; onChange: (next: SettingsValue) => void }) {
  const set = <K extends keyof SettingsValue>(k: K, v: SettingsValue[K]) => onChange({ ...value, [k]: v });
  return (
    <div className="grid gap-5 max-w-3xl">
      <Select label="Shell" value={value.shell} options={['zsh','bash','fish']} onChange={(v) => set('shell', v)} />
      <TextField label="Prompt template" mono value={value.prompt} onChange={(v) => set('prompt', v)} hint="tokens: {cluster} {ns} {provider} {ctx} {time}" />
      <TextField label="Working directory" mono value={value.cwd} onChange={(v) => set('cwd', v)} hint="empty = inherit" />
      <KVList label="Environment variables" hint={`${value.env.length} active`} items={value.env} onChange={(v) => set('env', v)} keyPlaceholder="KUBE_EDITOR" valuePlaceholder="nvim" />
      <KVList label="Aliases" hint={`${value.aliases.length} active`} items={value.aliases} onChange={(v) => set('aliases', v)} keyPlaceholder="k" valuePlaceholder="kubectl" />
    </div>
  );
}
```

### `src/ui/ClusterListSkeleton.tsx`

```tsx
import * as React from 'react';
export function ClusterListSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <ul className="animate-pulse" aria-hidden>
      {Array.from({ length: rows }).map((_, i) => (
        <li key={i} className="grid grid-cols-[1fr_auto_auto_auto] items-center gap-3 px-3 py-2 border-l-2 border-transparent">
          <div><div className="h-3.5 w-32 rounded bg-surface-2" /><div className="mt-1.5 h-2.5 w-20 rounded bg-surface-2" /></div>
          <div className="h-5 w-12 rounded-sm bg-surface-2" />
          <div className="h-3 w-10 rounded bg-surface-2" />
          <div className="h-5 w-20 rounded-sm bg-surface-2" />
        </li>
      ))}
    </ul>
  );
}
```

### `src/ui/ErrorBanner.tsx`

```tsx
import * as React from 'react';
import { IconError, IconRefresh } from '../icons/Icon';
import { Button } from './Button';
export interface ErrorBannerProps { title: string; body?: React.ReactNode; onRetry?: () => void; }
export function ErrorBanner({ title, body, onRetry }: ErrorBannerProps) {
  return (
    <div role="alert" className="m-3 flex gap-3 items-start p-3 rounded-md bg-error/10 border border-error/40">
      <IconError size={16} className="text-error mt-0.5 shrink-0"/>
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-text">{title}</div>
        {body && <div className="text-xs text-text-2 mt-0.5">{body}</div>}
      </div>
      {onRetry && (<Button variant="secondary" size="sm" leading={<IconRefresh size={12}/>} onClick={onRetry}>Retry</Button>)}
    </div>
  );
}
export const MgmtUnreachableBanner = ({ name, onRetry }: { name: string; onRetry?: () => void }) => (
  <ErrorBanner title="Management cluster unreachable" body={`Couldn't reach ${name}. Check VPN / SSO and try again.`} onRetry={onRetry}/>
);
export const KubeconfigUnavailableBanner = ({ cluster, onRetry }: { cluster: string; onRetry?: () => void }) => (
  <ErrorBanner title={`Kubeconfig unavailable for ${cluster}`} body="The mgmt cluster reported the workload cluster is reachable but its kubeconfig secret is missing or unreadable." onRetry={onRetry}/>
);
```

---

## 4) Layouts

`src/App.tsx` and `src/SettingsView.tsx` from the handoff are *skeletons*. Plan 5 rewrites them to wire to the real API (clusters via fetch+SSE, terminal via xterm.js+WS, settings via fetch+PUT, mgmt picker via fetch+PUT). The handoff versions used local mock state; the real ones use the API client in `src/api/`. The visual structure (header / left ClusterList sidebar / right TerminalPanel; settings = header + tabbed full-width form; narrow-window collapse below ~900px) carries over.

---

## 5) File map (handoff → repo)

```
internal/web/frontend/
  index.html
  package.json
  vite.config.ts
  tsconfig.json
  tailwind.config.js
  postcss.config.js
  src/
    main.tsx
    styles/tokens.css
    brand/KapishMark.tsx
    brand/KapishLockup.tsx
    icons/Icon.tsx
    ui/Button.tsx
    ui/AppHeader.tsx
    ui/FilterInput.tsx
    ui/PhaseChip.tsx
    ui/ClusterListRow.tsx
    ui/ClusterListSkeleton.tsx
    ui/TerminalPanel.tsx
    ui/EmptyState.tsx
    ui/ConfirmDialog.tsx
    ui/Toast.tsx
    ui/SettingsTabs.tsx
    ui/SettingsSectionForm.tsx
    ui/Field.tsx
    ui/KVList.tsx
    ui/ErrorBanner.tsx
    api/client.ts          # fetch wrappers: clusters, config, mgmts, sessions
    api/stream.ts          # SSE subscriber for /clusters/stream
    api/ws.ts              # WebSocket framing helper (0x00 data, 0x01 resize, 0x02 ping)
    components/MgmtPicker.tsx
    components/TerminalSession.tsx   # owns the xterm.js instance + WS lifecycle
    App.tsx
    SettingsView.tsx
  dist/                    # vite build output, committed so `go install` ships it
```

## 6) Notes for the Solid swap

- All components are pure-render functions; no `Suspense`, `useId`, `useTransition`, no portals from React internals (`ConfirmDialog` uses `position: fixed` + a top-level mount, not `createPortal`).
- Refs forwarded with `ref` only on `TerminalPanel`, where xterm.js needs the host DOM node.
- State is `useState` with primitives or arrays; replace with Solid `createSignal` 1:1.
