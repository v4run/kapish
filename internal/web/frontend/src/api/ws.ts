export const FRAME_DATA = 0x00;
export const FRAME_RESIZE = 0x01;
export const FRAME_PING = 0x02;

const enc = new TextEncoder();
const dec = new TextDecoder();

export function dataFrame(s: string): Uint8Array {
  const body = enc.encode(s);
  const out = new Uint8Array(body.length + 1);
  out[0] = FRAME_DATA;
  out.set(body, 1);
  return out;
}
export function resizeFrame(cols: number, rows: number): Uint8Array {
  const body = enc.encode(JSON.stringify({ cols, rows }));
  const out = new Uint8Array(body.length + 1);
  out[0] = FRAME_RESIZE;
  out.set(body, 1);
  return out;
}
export function pingFrame(): Uint8Array { return new Uint8Array([FRAME_PING]); }

// decodeIncoming returns the data payload string for a 0x00 frame, or null for
// pong / unknown frames.
export function decodeIncoming(buf: ArrayBuffer): string | null {
  const u = new Uint8Array(buf);
  if (u.length === 0) return null;
  if (u[0] === FRAME_DATA) return dec.decode(u.subarray(1));
  return null; // 0x02 pong etc.
}
