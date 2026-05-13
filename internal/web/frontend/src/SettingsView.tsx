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
