import { KapishMark } from './KapishMark';
export function KapishWordmark({ className = '', cursor = true }: { className?: string; cursor?: boolean }) {
  return (
    <span className={`inline-flex items-baseline font-bold leading-none ${className}`}>
      <span className="font-sans tracking-tight">kapi</span>
      <span className="font-mono text-primary ml-[0.04em]">sh</span>
      {cursor && (<span aria-hidden className="cursor inline-block bg-primary ml-[0.10em]" style={{ width: '0.42em', height: '0.78em', transform: 'translateY(0.04em)', borderRadius: 1 }}/>)}
    </span>
  );
}
export function KapishLockup({ size = 28 }: { size?: number }) {
  return (
    <div className="inline-flex items-center gap-3">
      <KapishMark size={Math.round(size * 1.05)} />
      <KapishWordmark className="text-text" />
    </div>
  );
}
