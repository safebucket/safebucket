import { Ellipsis } from "lucide-react";
import type { Row } from "@tanstack/react-table";

import { FileActions } from "@/components/FileActions/FileActions";
import { Button } from "@/components/ui/button";

interface DataTableRowActionsProps<TData> {
  row: Row<TData>;
}

export function DataTableRowActions<
  TData extends { id: string; name: string },
>({ row }: DataTableRowActionsProps<TData>) {
  return (
    <div onClick={(e) => e.stopPropagation()}>
      <FileActions file={row.original as any} type="dropdown">
        <Button
          variant="ghost"
          className="data-[state=open]:bg-muted flex h-8 w-8 p-0"
        >
          <Ellipsis className="h-4 w-4" />
          <span className="sr-only">Open file actions</span>
        </Button>
      </FileActions>
    </div>
  );
}
