"use client";

import React from "react";

import { Row } from "@tanstack/react-table";
import { Ellipsis } from "lucide-react";

import { FileActions } from "@/components/FileActions/FileActions";
import { IFile } from "@/components/bucket-view/helpers/types";
import { Button } from "@/components/ui/button";

interface DataTableRowActionsProps<TData> {
  row: Row<TData>;
}

export function DataTableRowActions<TData extends IFile>({
  row,
}: DataTableRowActionsProps<TData>) {
  return (
    <FileActions file={row.original} type="dropdown">
      <Button
        variant="ghost"
        className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
      >
        <Ellipsis className="h-4 w-4" />
        <span className="sr-only">Open file actions</span>
      </Button>
    </FileActions>
  );
}
