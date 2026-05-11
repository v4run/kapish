export function ClusterListSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <ul className="animate-pulse" aria-hidden>
      {Array.from({ length: rows }).map((_, i) => (
        <li key={i} className="grid grid-cols-[1fr_auto_auto_auto] items-center gap-3 px-3 py-2 border-l-2 border-transparent">
          <div><div className="h-3.5 w-32 rounded bg-surface-2" /><div className="mt-1.5 h-2.5 w-20 rounded bg-surface-2" /></div>
          <div className="h-5 w-12 rounded-sm bg-surface-2" />
          <div className="h-3 w-10 rounded bg-surface-2" />
          <div className="h-5 w-20 rounded-sm bg-surface-2" />
        </li>
      ))}
    </ul>
  );
}
