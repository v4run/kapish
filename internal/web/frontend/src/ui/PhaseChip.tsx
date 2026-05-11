export type Phase = 'Provisioned' | 'Provisioning' | 'Pending' | 'Failed' | 'Deleting';
const PHASE: Record<Phase, { color: string; bg: string; label: string }> = {
  Provisioned:  { color: 'text-success', bg: 'bg-success/15',  label: 'Provisioned' },
  Provisioning: { color: 'text-warning', bg: 'bg-warning/15',  label: 'Provisioning' },
  Pending:      { color: 'text-info',    bg: 'bg-info/15',     label: 'Pending' },
  Failed:       { color: 'text-error',   bg: 'bg-error/15',    label: 'Failed' },
  Deleting:     { color: 'text-muted',   bg: 'bg-surface-2',   label: 'Deleting' },
};
// Tolerate unknown/empty phase strings by falling back to a neutral chip.
export function PhaseChip({ phase }: { phase: string }) {
  const p = PHASE[phase as Phase] ?? { color: 'text-muted', bg: 'bg-surface-2', label: phase || 'Unknown' };
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 h-5 rounded-sm font-mono text-2xs font-medium ${p.color} ${p.bg}`}>
      <span className="size-1.5 rounded-full bg-current" />{p.label}
    </span>
  );
}
