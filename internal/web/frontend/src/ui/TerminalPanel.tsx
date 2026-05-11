import * as React from 'react';
import { Button } from './Button';
import { IconPower, IconTerminal } from '../icons/Icon';
import { PhaseChip } from './PhaseChip';
export interface TerminalPanelProps {
  cluster: string; namespace?: string; phase?: string;
  terminalRef: React.Ref<HTMLDivElement>; onDisconnect?: () => void; toolbarExtra?: React.ReactNode;
}
export function TerminalPanel({ cluster, namespace, phase, terminalRef, onDisconnect, toolbarExtra }: TerminalPanelProps) {
  return (
    <section className="flex-1 min-w-0 flex flex-col bg-bg">
      <div className="h-10 flex-shrink-0 flex items-center gap-3 px-4 border-b border-border bg-bg-2">
        <IconTerminal size={14} className="text-muted" />
        <span className="font-mono text-sm font-medium text-text truncate">{cluster}</span>
        {namespace && <span className="font-mono text-xs text-muted">· {namespace}</span>}
        {phase && <PhaseChip phase={phase} />}
        <div className="flex-1" />
        {toolbarExtra}
        <Button variant="icon" size="sm" onClick={onDisconnect} aria-label="Disconnect" className="hover:text-error"><IconPower size={14} /></Button>
      </div>
      <div ref={terminalRef} className="flex-1 min-h-0 bg-[#08090c] font-mono" data-terminal />
    </section>
  );
}
