import * as React from 'react';
type Variant = 'primary' | 'secondary' | 'icon' | 'danger';
type Size = 'sm' | 'md';
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant; size?: Size; leading?: React.ReactNode; trailing?: React.ReactNode;
}
const base = 'inline-flex items-center justify-center gap-2 font-medium rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus-visible:shadow-focus';
const sizes: Record<Size, string> = { sm: 'h-7 px-2.5 text-xs', md: 'h-9 px-3.5 text-sm' };
const variants: Record<Variant, string> = {
  primary: 'bg-primary text-bg hover:brightness-110 active:brightness-95',
  secondary: 'bg-surface text-text-2 border border-border hover:bg-surface-2 hover:text-text',
  icon: 'bg-transparent text-text-2 hover:bg-surface-2 hover:text-text aspect-square px-0',
  danger: 'bg-transparent text-error border border-error/50 hover:bg-error/10',
};
export function Button({ variant = 'secondary', size = 'md', leading, trailing, children, className = '', ...rest }: ButtonProps) {
  return (<button {...rest} className={`${base} ${sizes[size]} ${variants[variant]} ${className}`}>{leading}{children}{trailing}</button>);
}
