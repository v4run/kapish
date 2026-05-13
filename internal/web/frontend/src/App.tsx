import * as React from 'react';
import { AppHeader } from './ui/AppHeader';
import { FilterInput } from './ui/FilterInput';
import { ClusterListRow } from './ui/ClusterListRow';
import { ClusterListSkeleton } from './ui/ClusterListSkeleton';
import { SelectClusterEmpty, NoClustersFoundEmpty } from './ui/EmptyState';
import { ConfirmDialog } from './ui/ConfirmDialog';
import { Toast, ToastStack } from './ui/Toast';
import { ErrorBanner } from './ui/ErrorBanner';
import { MgmtPicker } from './components/MgmtPicker';
import { TerminalSession } from './components/TerminalSession';
import { SettingsView } from './SettingsView';
import { getClusters, getMgmts } from './api/client';
import { subscribeClusterStream } from './api/stream';
import type { Cluster } from './api/types';

type View = 'main' | 'settings';

export default function App() {
  const [clusters, setClusters] = React.useState<Cluster[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [loadErr, setLoadErr] = React.useState<string | null>(null);
  const [filter, setFilter] = React.useState('');
  const [selectedKey, setSelectedKey] = React.useState<string | null>(null);
  const [pendingCluster, setPendingCluster] = React.useState<Cluster | null>(null); // confirm dialog target
  const [confirmFailed, setConfirmFailed] = React.useState<Cluster | null>(null);
  const [view, setView] = React.useState<View>('main');
  const [mgmtPickerOpen, setMgmtPickerOpen] = React.useState(false);
  const [mgmtCluster, setMgmtCluster] = React.useState('—');
  const [toasts, setToasts] = React.useState<{ id: number; tone: 'error' | 'info'; title: string }[]>([]);
  const toastId = React.useRef(0);
  const pushToast = React.useCallback((tone: 'error' | 'info', title: string) => {
    const id = ++toastId.current;
    setToasts((t) => [...t, { id, tone, title }]);
    window.setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 5000);
  }, []);

  const handleDisconnect = React.useCallback(() => setSelectedKey(null), []);
  const handleTerminalError = React.useCallback((m: string) => pushToast('error', m), [pushToast]);

  const refresh = React.useCallback(() => {
    setLoading(true);
    getClusters().then((cs) => { setClusters(cs); setLoadErr(null); }).catch((e) => setLoadErr(e instanceof Error ? e.message : String(e))).finally(() => setLoading(false));
  }, []);

  React.useEffect(() => { refresh(); getMgmts().then((m) => setMgmtCluster(m.current || '—')).catch(() => {}); }, [refresh]);

  React.useEffect(() => {
    const close = subscribeClusterStream({
      onSync: () => refresh(),
      onCluster: (type, c) => {
        setClusters((prev) => {
          const key = c.namespace + '/' + c.name;
          const without = prev.filter((p) => p.namespace + '/' + p.name !== key);
          if (type === 'deleted') return without;
          return [...without, c].sort((a, b) => (a.namespace !== b.namespace ? a.namespace.localeCompare(b.namespace) : a.name.localeCompare(b.name)));
        });
      },
    });
    return close;
  }, [refresh]);

  const sorted = React.useMemo(
    () => [...clusters].sort((a, b) => (a.namespace !== b.namespace ? a.namespace.localeCompare(b.namespace) : a.name.localeCompare(b.name))),
    [clusters],
  );
  const filtered = React.useMemo(
    () => (!filter ? sorted : sorted.filter((c) => c.name.includes(filter) || c.namespace.includes(filter))),
    [sorted, filter],
  );
  const keyOf = (c: Cluster) => c.namespace + '/' + c.name;
  const active = clusters.find((c) => keyOf(c) === selectedKey) ?? null;

  const tryConnect = (c: Cluster) => {
    if (c.phase === 'Failed' || c.phase === 'Deleting') { setConfirmFailed(c); return; }
    doConnect(c);
  };
  const doConnect = (c: Cluster) => {
    if (active && keyOf(active) !== keyOf(c)) { setPendingCluster(c); return; } // confirm replace
    setSelectedKey(keyOf(c));
  };

  if (view === 'settings') {
    return <SettingsView mgmtCluster={mgmtCluster} onClose={() => setView('main')} />;
  }

  return (
    <div className="h-screen w-screen flex flex-col bg-bg text-text font-sans relative">
      <AppHeader
        mgmtCluster={mgmtCluster}
        onPickMgmt={() => setMgmtPickerOpen((v) => !v)}
        onRefresh={refresh}
        refreshing={loading}
        onSettings={() => setView('settings')}
      />
      {mgmtPickerOpen && (
        <MgmtPicker onClose={() => setMgmtPickerOpen(false)} onSwitched={(name) => { setMgmtCluster(name); setSelectedKey(null); refresh(); }} />
      )}
      <div className="flex-1 min-h-0 flex">
        <aside className="flex-shrink-0 w-[340px] border-r border-border bg-bg-2 flex flex-col">
          <div className="p-3 border-b border-border">
            <FilterInput value={filter} onChange={setFilter} hint={`${filtered.length} matches`} />
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto">
            {loadErr && <ErrorBanner title="Couldn't load clusters" body={loadErr} onRetry={refresh} />}
            {loading && clusters.length === 0 && !loadErr && <ClusterListSkeleton />}
            {!loading && !loadErr && filtered.length === 0 && <NoClustersFoundEmpty onClear={() => setFilter('')} />}
            {filtered.map((c) => (
              <ClusterListRow key={keyOf(c)} name={c.name} namespace={c.namespace} phase={c.phase} version={c.version} provider={c.provider}
                selected={selectedKey === keyOf(c)} onClick={() => tryConnect(c)} onConnect={() => tryConnect(c)} />
            ))}
          </div>
        </aside>
        {active ? (
          <TerminalSession namespace={active.namespace} cluster={active.name} phase={active.phase}
            onDisconnect={handleDisconnect} onError={handleTerminalError} />
        ) : (
          <div className="flex-1"><SelectClusterEmpty /></div>
        )}
      </div>

      <ConfirmDialog open={!!confirmFailed} title={confirmFailed ? `${confirmFailed.name} is ${confirmFailed.phase}` : ''}
        body="Some kubectl calls may hang. Spawn a shell anyway?" confirmLabel="Continue anyway"
        onConfirm={() => { const c = confirmFailed!; setConfirmFailed(null); doConnect(c); }} onCancel={() => setConfirmFailed(null)} />

      <ConfirmDialog open={!!pendingCluster} title="Disconnect current shell?"
        body="The kubeconfig for the current session is discarded." confirmLabel="Disconnect" tone="danger"
        onConfirm={() => { const c = pendingCluster!; setPendingCluster(null); setSelectedKey(keyOf(c)); }} onCancel={() => setPendingCluster(null)} />

      <ToastStack>
        {toasts.map((t) => <Toast key={t.id} tone={t.tone} title={t.title} onClose={() => setToasts((x) => x.filter((y) => y.id !== t.id))} />)}
      </ToastStack>
    </div>
  );
}
