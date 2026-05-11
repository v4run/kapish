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
