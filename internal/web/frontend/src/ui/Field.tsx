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
