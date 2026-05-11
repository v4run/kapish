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
