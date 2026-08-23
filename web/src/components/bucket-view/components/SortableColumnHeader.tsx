import { ArrowDown, ArrowUp, ChevronsUpDown } from "lucide-react";
import type { FC } from "react";
import { cn } from "@/lib/utils";

interface ISortableColumnHeaderProps {
  label: string;
  active: boolean;
  dir: "asc" | "desc";
  onClick: () => void;
  className?: string;
}

export const SortableColumnHeader: FC<ISortableColumnHeaderProps> = ({
  label,
  active,
  dir,
  onClick,
  className,
}: ISortableColumnHeaderProps) => (
  <button
    type="button"
    onClick={onClick}
    className={cn(
      "text-muted-foreground hover:text-foreground -ml-1 flex items-center gap-1 rounded px-1 text-xs font-medium tracking-wide uppercase transition-colors",
      className,
    )}
  >
    {label}
    {active ? (
      dir === "asc" ? (
        <ArrowUp className="h-3 w-3" />
      ) : (
        <ArrowDown className="h-3 w-3" />
      )
    ) : (
      <ChevronsUpDown className="h-3 w-3 opacity-50" />
    )}
  </button>
);
