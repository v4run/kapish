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
