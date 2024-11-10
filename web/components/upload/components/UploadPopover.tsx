import React, { FC } from "react";

import { ChevronDownIcon, CircleCheck, FileIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Progress } from "@/components/ui/progress";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

export const UploadPopover: FC = () => {
  const { uploads } = useUploadContext();

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" className="relative">
          Uploads
          {uploads.length > 0 && (
            <Badge className="absolute -right-2 -top-2 h-6 w-6 justify-center">
              {uploads.length}
            </Badge>
          )}
          <ChevronDownIcon className="ml-2 h-4 w-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80">
        {!uploads.length && (
          <p className="flex items-center justify-center">
            No uploads in progress
          </p>
        )}
        {uploads.map((upload) => (
          <div key={upload.id} className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <FileIcon className="h-5 w-5 text-muted-foreground" />
              <div className="text-sm font-medium">{upload.name}</div>
            </div>

            <Progress value={upload.progress} className="w-24" />

            {upload.progress == 100 ? (
              <CircleCheck className="text-primary" />
            ) : (
              <p>{upload.progress}%</p>
            )}
          </div>
        ))}
      </PopoverContent>
    </Popover>
  );
};
