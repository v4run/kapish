import { KapishLockup } from '../brand/KapishLockup';
import { Button } from './Button';
import { IconRefresh, IconSettings, IconChevron } from '../icons/Icon';
export interface AppHeaderProps {
  mgmtCluster: string; onPickMgmt?: () => void; onRefresh?: () => void; onSettings?: () => void; refreshing?: boolean; version?: string;
}
export function AppHeader({ mgmtCluster, onPickMgmt, onRefresh, onSettings, refreshing, version }: AppHeaderProps) {
  return (
    <header className="h-12 flex-shrink-0 flex items-center gap-4 px-4 border-b border-border bg-bg-2">
      <KapishLockup size={22} />
      {version && <span className="text-dim font-mono text-2xs">{version}</span>}
      <button onClick={onPickMgmt} className="ml-3 inline-flex items-center gap-2 px-2.5 h-7 rounded-md bg-surface border border-border text-text-2 hover:bg-surface-2 hover:text-text font-mono text-xs">
        <span className="size-1.5 rounded-full bg-success" />
        <span className="text-muted">mgmt</span>
        <span>{mgmtCluster}</span>
        <IconChevron size={12} className="text-muted" />
      </button>
      <div className="flex-1" />
      <Button variant="icon" size="sm" onClick={onRefresh} aria-label="Refresh"><IconRefresh size={14} className={refreshing ? 'spin' : ''} /></Button>
      <Button variant="icon" size="sm" onClick={onSettings} aria-label="Settings"><IconSettings size={14} /></Button>
    </header>
  );
}
