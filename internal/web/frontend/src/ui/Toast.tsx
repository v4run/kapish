import * as React from 'react';
import { IconCheck, IconWarn, IconInfo, IconError, IconClose } from '../icons/Icon';
type Tone = 'success' | 'warning' | 'error' | 'info';
const ICON: Record<Tone, React.FC<any>> = { success: IconCheck, warning: IconWarn, error: IconError, info: IconInfo };
const TONE: Record<Tone, string> = {
  success: 'text-success border-success/30', warning: 'text-warning border-warning/30', error: 'text-error border-error/40', info: 'text-info border-info/30',
};
export interface ToastProps { tone?: Tone; title: string; body?: React.ReactNode; onClose?: () => void; }
export function Toast({ tone = 'info', title, body, onClose }: ToastProps) {
  const Icon = ICON[tone];
  return (
    <div role="status" className={`flex items-start gap-3 w-[360px] p-3 pr-2 rounded-lg bg-surface border ${TONE[tone]} shadow-md`}>
      <Icon size={16} className="mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-text">{title}</div>
        {body && <div className="text-xs text-text-2 mt-0.5">{body}</div>}
      </div>
      {onClose && (<button onClick={onClose} className="text-muted hover:text-text"><IconClose size={12} /></button>)}
    </div>
  );
}
export function ToastStack({ children }: { children: React.ReactNode }) {
  return <div className="fixed bottom-4 right-4 z-40 flex flex-col gap-2">{children}</div>;
}
