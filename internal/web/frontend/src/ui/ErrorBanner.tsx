import * as React from 'react';
import { IconError, IconRefresh } from '../icons/Icon';
import { Button } from './Button';
export interface ErrorBannerProps { title: string; body?: React.ReactNode; onRetry?: () => void; }
export function ErrorBanner({ title, body, onRetry }: ErrorBannerProps) {
  return (
    <div role="alert" className="m-3 flex gap-3 items-start p-3 rounded-md bg-error/10 border border-error/40">
      <IconError size={16} className="text-error mt-0.5 shrink-0"/>
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-text">{title}</div>
        {body && <div className="text-xs text-text-2 mt-0.5">{body}</div>}
      </div>
      {onRetry && (<Button variant="secondary" size="sm" leading={<IconRefresh size={12}/>} onClick={onRetry}>Retry</Button>)}
    </div>
  );
}
export const MgmtUnreachableBanner = ({ name, onRetry }: { name: string; onRetry?: () => void }) => (
  <ErrorBanner title="Management cluster unreachable" body={`Couldn't reach ${name}. Check VPN / SSO and try again.`} onRetry={onRetry}/>
);
export const KubeconfigUnavailableBanner = ({ cluster, onRetry }: { cluster: string; onRetry?: () => void }) => (
  <ErrorBanner title={`Kubeconfig unavailable for ${cluster}`} body="The mgmt cluster reported the workload cluster is reachable but its kubeconfig secret is missing or unreadable." onRetry={onRetry}/>
);
