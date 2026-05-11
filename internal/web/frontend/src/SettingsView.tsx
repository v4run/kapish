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
            <Select label="Theme" value={cfg.ui.theme as 'dark' | 'light'} options={['dark', 'light']} onChange={() => toggleTheme()} />
          )}
          {val && tab !== 'theme' && <SettingsSectionForm value={val} onChange={applyForm} />}
        </main>
      </div>
    </div>
  );
}
