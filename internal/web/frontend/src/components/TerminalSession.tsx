import * as React from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { createSession } from '../api/client';
import { dataFrame, resizeFrame, pingFrame, decodeIncoming } from '../api/ws';
import { TerminalPanel } from '../ui/TerminalPanel';

export interface TerminalSessionProps {
  namespace: string;
  cluster: string;
  phase?: string;
  onDisconnect: () => void;
  onError?: (msg: string) => void;
}

export function TerminalSession({ namespace, cluster, phase, onDisconnect, onError }: TerminalSessionProps) {
  const hostRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    let disposed = false;
    const term = new Terminal({
      fontFamily: '"Geist Mono", ui-monospace, monospace',
      fontSize: 13,
      theme: { background: '#08090c' },
      cursorBlink: true,
      convertEol: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    fit.fit();

    let ws: WebSocket | null = null;
    let pingTimer: number | undefined;

    (async () => {
      try {
        const sess = await createSession(namespace, cluster);
        if (disposed) return;
        const wsURL = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + sess.wsUrl + '?token=' + encodeURIComponent(sess.wsToken);
        ws = new WebSocket(wsURL);
        ws.binaryType = 'arraybuffer';
        ws.onopen = () => {
          ws!.send(resizeFrame(term.cols, term.rows));
          pingTimer = window.setInterval(() => { try { ws?.send(pingFrame()); } catch { /* ignore */ } }, 30000);
        };
        ws.onmessage = (ev) => {
          const s = typeof ev.data === 'string' ? null : decodeIncoming(ev.data as ArrayBuffer);
          if (s) term.write(s);
        };
        ws.onclose = () => { if (!disposed) onDisconnect(); };
        ws.onerror = () => { if (!disposed && onError) onError('terminal connection error'); };
        term.onData((d) => { try { ws?.send(dataFrame(d)); } catch { /* ignore */ } });
      } catch (e) {
        if (!disposed && onError) onError(e instanceof Error ? e.message : String(e));
        if (!disposed) onDisconnect();
      }
    })();

    const onResize = () => {
      try {
        fit.fit();
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(resizeFrame(term.cols, term.rows));
      } catch { /* ignore */ }
    };
    window.addEventListener('resize', onResize);

    return () => {
      disposed = true;
      window.removeEventListener('resize', onResize);
      if (pingTimer) window.clearInterval(pingTimer);
      try { ws?.close(); } catch { /* ignore */ }
      term.dispose();
    };
  }, [namespace, cluster, onDisconnect, onError]);

  return <TerminalPanel cluster={cluster} namespace={namespace} phase={phase} terminalRef={hostRef} onDisconnect={onDisconnect} />;
}
