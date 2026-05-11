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
