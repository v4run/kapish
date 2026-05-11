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
