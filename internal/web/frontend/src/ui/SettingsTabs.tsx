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
