/**
 * PageHeader — a large, calm page title with optional subtitle and a trailing
 * action slot. One header per screen; sets the visual hierarchy.
 */
export interface PageHeaderProps {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
}

export function PageHeader({ title, subtitle, action }: PageHeaderProps) {
  return (
    <header className="mb-7 flex items-start justify-between gap-4">
      <div className="min-w-0">
        <h1 className="truncate text-[1.7rem] font-semibold leading-tight tracking-tight text-foreground">
          {title}
        </h1>
        {subtitle && <p className="mt-1.5 text-sm text-muted">{subtitle}</p>}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </header>
  );
}
