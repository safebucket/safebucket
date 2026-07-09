import type { ReactNode } from "react";

interface SettingRowProps {
  label: string;
  children: ReactNode;
}

export function SettingRow({ label, children }: SettingRowProps) {
  return (
    <div className="flex items-start justify-between gap-4 py-2 text-sm">
      <span className="text-muted-foreground shrink-0">{label}</span>
      <div className="flex flex-wrap items-center justify-end gap-1 break-all text-right font-medium">
        {children}
      </div>
    </div>
  );
}
