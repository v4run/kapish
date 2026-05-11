import * as React from 'react';
const I = ({ size = 16, className = '', children, label }: { size?: number; className?: string; children: React.ReactNode; label: string }) => (
  <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" className={className} role="img" aria-label={label}>{children}</svg>
);
export const IconSearch  = (p: any) => <I {...p} label="search"><circle cx="7" cy="7" r="4.5"/><path d="M11 11l3 3"/></I>;
export const IconClose   = (p: any) => <I {...p} label="close"><path d="M3.5 3.5l9 9M12.5 3.5l-9 9"/></I>;
export const IconRefresh = (p: any) => <I {...p} label="refresh"><path d="M13.5 3.5v3h-3"/><path d="M13 6.5A5 5 0 1 0 13.5 11"/></I>;
export const IconSettings= (p: any) => <I {...p} label="settings"><circle cx="8" cy="8" r="2"/><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.5 3.5l1.4 1.4M11.1 11.1l1.4 1.4M3.5 12.5l1.4-1.4M11.1 4.9l1.4-1.4"/></I>;
export const IconChevron = (p: any) => <I {...p} label="chevron"><path d="M6 4l4 4-4 4"/></I>;
export const IconPower   = (p: any) => <I {...p} label="disconnect"><path d="M5 4a4.5 4.5 0 1 0 6 0"/><path d="M8 1.5v6"/></I>;
export const IconCheck   = (p: any) => <I {...p} label="check"><path d="M3 8.5l3 3 7-7"/></I>;
export const IconWarn    = (p: any) => <I {...p} label="warning"><path d="M8 3l6 10H2z"/><path d="M8 7v3M8 11.5v.01"/></I>;
export const IconInfo    = (p: any) => <I {...p} label="info"><circle cx="8" cy="8" r="6.5"/><path d="M8 7.5v3.5M8 5v.01"/></I>;
export const IconError   = (p: any) => <I {...p} label="error"><circle cx="8" cy="8" r="6.5"/><path d="M5.5 5.5l5 5M10.5 5.5l-5 5"/></I>;
export const IconPlus    = (p: any) => <I {...p} label="add"><path d="M8 3v10M3 8h10"/></I>;
export const IconTrash   = (p: any) => <I {...p} label="remove"><path d="M3 4.5h10M6 4.5V3h4v1.5M5 4.5v8a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-8"/></I>;
export const IconTerminal= (p: any) => <I {...p} label="terminal"><rect x="1.5" y="2.5" width="13" height="11" rx="1.5"/><path d="M4 6l2 2-2 2M8 10h4"/></I>;
