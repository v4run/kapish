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
