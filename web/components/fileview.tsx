import React from "react";

import { cn } from "@/lib/utils";
import { FileTypeIcon } from "lucide-react";

import { Card } from "@/components/ui/card";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";

export function FileView({ file }) {
  return (
    <div className={cn("space-y-3")}>
      <ContextMenu>
        <ContextMenuTrigger>
          <Card
            key={file.id}
            className={`flex flex-col gap-4 p-4 ${file.selected ? "bg-primary text-primary-foreground" : ""}`}
          >
            <div className="flex items-center gap-4">
              <div
                className={`flex aspect-square w-12 items-center justify-center rounded-md bg-muted ${
                  file.selected ? "bg-primary-foreground text-primary" : ""
                }`}
              >
                <FileTypeIcon className="h-6 w-6" />
              </div>
              <div className="flex-1">
                <h3
                  className={`truncate font-medium ${file.selected ? "text-primary-foreground" : ""}`}
                >
                  {file.name}
                </h3>
                <p
                  className={`text-sm ${file.selected ? "text-primary-foreground" : "text-muted-foreground"}`}
                >
                  {file.size}
                </p>
              </div>
            </div>
            <div
              className={`text-sm ${file.selected ? "text-primary-foreground" : "text-muted-foreground"}`}
            >
              Modified: {file.modified}
            </div>
          </Card>
        </ContextMenuTrigger>
        <ContextMenuContent className="w-40">
          <ContextMenuItem>Play Next</ContextMenuItem>
          <ContextMenuItem>Play Later</ContextMenuItem>
          <ContextMenuItem>Create Station</ContextMenuItem>
          <ContextMenuSeparator />
          <ContextMenuItem>Like</ContextMenuItem>
          <ContextMenuItem>Share</ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
    </div>
  );
}
