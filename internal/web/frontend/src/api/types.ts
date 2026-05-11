export interface Cluster {
  name: string;
  namespace: string;
  phase: string;
  controlPlaneReady: boolean;
  infrastructureReady: boolean;
  version: string;
  provider: string;
  ageSeconds: number;
}
export interface MgmtEntry { name: string; context?: string; namespace?: string }
export interface Mgmts { current: string; entries: MgmtEntry[] }
export interface CreateSessionResp { sessionId: string; wsUrl: string; wsToken: string }
// Config mirrors kconfig.Config's JSON (lowercase keys).
export interface KapishConfig {
  managementClusters: { current?: string; entries?: MgmtEntry[] };
  shell: { command?: string; cwd?: string; env?: Record<string, string>; aliases?: Record<string, string>; prompt?: string };
  ui: { theme: string; refreshIntervalSec: number; oneShot: boolean };
  web: { defaultPort: number; openBrowser: boolean; bindAddr: string };
}
