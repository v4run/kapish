import type { Cluster } from './types';

export interface ClusterStreamHandlers {
  onSync: () => void; // re-fetch the snapshot
  onCluster: (type: 'added' | 'modified' | 'deleted', cluster: Cluster) => void;
  onError?: (e: Event) => void;
}

// subscribeClusterStream opens an EventSource on /api/v1/clusters/stream and
// dispatches to handlers. Returns a close() func. EventSource auto-reconnects
// on transport errors; on reconnect the server sends `event: sync` again, so
// onSync should re-fetch the snapshot.
export function subscribeClusterStream(h: ClusterStreamHandlers): () => void {
  const es = new EventSource('/api/v1/clusters/stream');
  es.addEventListener('sync', () => h.onSync());
  es.addEventListener('cluster', (ev) => {
    try {
      const m = JSON.parse((ev as MessageEvent).data) as { type: 'added' | 'modified' | 'deleted'; cluster: Cluster };
      h.onCluster(m.type, m.cluster);
    } catch { /* ignore malformed */ }
  });
  if (h.onError) es.onerror = h.onError;
  return () => es.close();
}
