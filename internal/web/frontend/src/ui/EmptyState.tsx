import * as React from 'react';
import { KapishMark } from '../brand/KapishMark';
export interface EmptyStateProps { icon?: React.ReactNode; title: string; body?: string; action?: React.ReactNode; }
export function EmptyState({ icon, title, body, action }: EmptyStateProps) {
  return (
    <div className="h-full flex flex-col items-center justify-center px-8 text-center">
      {icon && <div className="text-muted mb-3">{icon}</div>}
      <div className="text-text text-md font-medium">{title}</div>
      {body && <p className="mt-1 text-sm text-text-2 max-w-xs">{body}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}
export const SelectClusterEmpty = () => (
  <EmptyState
    icon={<KapishMark size={36} />}
    title="Select a cluster to start a shell"
    body="Pick any cluster from the list. kapish will fetch its kubeconfig and spawn your configured shell." />
);
export const NoClustersFoundEmpty = ({ onClear }: { onClear?: () => void }) => (
  <EmptyState title="No clusters found" body="No clusters match your filters. Try clearing the search or switching management contexts."
    action={onClear && <button onClick={onClear} className="text-primary text-sm hover:underline">Clear filters</button>} />
);
