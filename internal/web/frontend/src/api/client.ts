import type { Cluster, Mgmts, CreateSessionResp, KapishConfig } from './types';

async function jsonOrThrow<T>(r: Response): Promise<T> {
  if (!r.ok) {
    let msg = `${r.status} ${r.statusText}`;
    try { const b = await r.json(); if (b && b.error) msg = b.error; } catch { /* ignore */ }
    throw new Error(msg);
  }
  return r.json() as Promise<T>;
}

export async function getClusters(): Promise<Cluster[]> {
  const r = await fetch('/api/v1/clusters');
  const body = await jsonOrThrow<{ clusters: Cluster[] }>(r);
  return body.clusters ?? [];
}
export async function getConfig(): Promise<KapishConfig> {
  return jsonOrThrow<KapishConfig>(await fetch('/api/v1/config'));
}
export async function putConfig(cfg: KapishConfig): Promise<void> {
  const r = await fetch('/api/v1/config', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg) });
  await jsonOrThrow<{ status: string }>(r);
}
export async function getMgmts(): Promise<Mgmts> {
  return jsonOrThrow<Mgmts>(await fetch('/api/v1/mgmts'));
}
export async function putMgmtsCurrent(name: string): Promise<void> {
  const r = await fetch('/api/v1/mgmts/current', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name }) });
  await jsonOrThrow<{ status: string }>(r);
}
export async function createSession(namespace: string, cluster: string): Promise<CreateSessionResp> {
  const r = await fetch('/api/v1/sessions', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ namespace, cluster }) });
  return jsonOrThrow<CreateSessionResp>(r);
}
