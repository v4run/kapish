import { PhaseChip } from './PhaseChip';
const PROVIDER_TINT: Record<string, string> = {
  aws: 'text-p-aws bg-p-aws/15', gcp: 'text-p-gcp bg-p-gcp/15', azure: 'text-p-azure bg-p-azure/15',
  vsphere: 'text-p-vsphere bg-p-vsphere/15', hetzner: 'text-p-hetzner bg-p-hetzner/15',
};
export interface ClusterListRowProps {
  name: string; namespace: string; phase: string; version: string; provider: string;
  selected?: boolean; onClick?: () => void; onConnect?: () => void;
}
export function ClusterListRow({ name, namespace, phase, version, provider, selected, onClick, onConnect }: ClusterListRowProps) {
  const tint = PROVIDER_TINT[provider] ?? 'text-muted bg-surface-2';
  return (
    <button onClick={onClick} onDoubleClick={onConnect}
      className={`w-full text-left grid items-center gap-3 grid-cols-[1fr_auto_auto_auto] px-3 py-2 border-l-2 ${selected ? 'bg-accent/12 border-accent text-text' : 'border-transparent text-text-2 hover:bg-surface-2'} transition-colors`}>
      <div className="min-w-0">
        <div className="font-mono text-sm font-medium text-text truncate">{name}</div>
        <div className="text-2xs text-muted font-mono truncate">{namespace}</div>
      </div>
      <span className={`inline-flex items-center gap-1.5 px-2 h-5 rounded-sm font-mono text-2xs ${tint}`}><span className="size-1.5 rounded-full bg-current" />{provider || '-'}</span>
      <span className="font-mono text-2xs text-muted">{version || '-'}</span>
      <PhaseChip phase={phase} />
    </button>
  );
}
