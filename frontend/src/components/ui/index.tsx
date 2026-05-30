export function Spinner({ className = '' }: { className?: string }) {
  return (
    <div
      className={`inline-block h-5 w-5 animate-spin rounded-full border-2 border-gray-300 border-t-blue-600 ${className}`}
    />
  );
}

export function Button({
  children,
  variant = 'primary',
  className = '',
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'ghost' | 'danger' }) {
  const base =
    'inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50';
  const variants = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700',
    ghost: 'text-gray-600 hover:bg-gray-100',
    danger: 'bg-red-600 text-white hover:bg-red-700',
  };
  return <button className={`${base} ${variants[variant]} ${className}`} {...props}>{children}</button>;
}

export function Card({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return <div className={`rounded-xl border border-gray-200 bg-white p-4 shadow-sm ${className}`}>{children}</div>;
}

export function Badge({ children, color = 'gray' }: { children: React.ReactNode; color?: 'gray' | 'green' | 'red' | 'blue' }) {
  const colors = {
    gray: 'bg-gray-100 text-gray-700',
    green: 'bg-green-100 text-green-700',
    red: 'bg-red-100 text-red-700',
    blue: 'bg-blue-100 text-blue-700',
  };
  return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${colors[color]}`}>{children}</span>;
}
