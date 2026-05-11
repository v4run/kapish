type Props = { size?: number; className?: string; accent?: string; violet?: string; mono?: boolean };
export function KapishMark({ size = 32, className, accent = 'currentColor', violet, mono = false }: Props) {
  const A = accent;
  const V = mono ? accent : (violet ?? 'rgb(180 141 255)');
  const stroke = 2.6;
  const dot = 2.8;
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" className={className} aria-label="kapish">
      <line x1="9" y1="6" x2="9" y2="26" stroke={A} strokeWidth={stroke} strokeLinecap="round"/>
      <line x1="9" y1="16" x2="20" y2="7" stroke={V} strokeWidth={stroke} strokeLinecap="round"/>
      <line x1="9" y1="16" x2="24" y2="26" stroke={V} strokeWidth={stroke} strokeLinecap="round"/>
      <circle cx="9" cy="6" r={dot} fill={A}/>
      <circle cx="9" cy="26" r={dot} fill={A}/>
      <circle cx="9" cy="16" r={dot * 0.85} fill={A}/>
      <circle cx="20" cy="7" r={dot} fill={V}/>
      <circle cx="24" cy="26" r={dot} fill={V}/>
    </svg>
  );
}
