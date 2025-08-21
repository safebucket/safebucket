import React, { FC } from "react";

import { ChevronDownIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Progress } from "@/components/ui/progress";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";
import { UploadStatus } from "@/components/upload/helpers/types";
import { getStatusIcon, getStatusText } from "@/components/upload/helpers/utils";


export const UploadPopover: FC = () => {
  const { uploads } = useUploadContext();
  
  const activeUploads = uploads.filter(upload => upload.status !== UploadStatus.success);
  const completedCount = uploads.filter(upload => upload.status === UploadStatus.success).length;
  const failedCount = uploads.filter(upload => upload.status === UploadStatus.failed).length;

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
      <PopoverContent className="w-96 max-h-80 overflow-y-auto">
        {!uploads.length && (
          <p className="flex items-center justify-center text-muted-foreground py-4">
            No uploads in progress
          </p>
        )}
        
        {uploads.length > 0 && (
          <div className="space-y-2">
            {(completedCount > 0 || failedCount > 0) && (
              <div className="flex items-center justify-between p-2 bg-muted/50 rounded text-xs">
                <span>
                  {completedCount > 0 && (
                    <span className="text-green-600">{completedCount} completed</span>
                  )}
                  {completedCount > 0 && failedCount > 0 && " • "}
                  {failedCount > 0 && (
                    <span className="text-red-600">{failedCount} failed</span>
                  )}
                </span>
                <span className="text-muted-foreground">
                  {activeUploads.length} active
                </span>
              </div>
            )}
            
            {uploads.map((upload) => (
              <div key={upload.id} className="flex items-center gap-3 p-2 rounded hover:bg-muted/30">
                <div className="flex-shrink-0">
                  {getStatusIcon(upload.status, upload.progress)}
                </div>
                
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate" title={upload.path}>
                    {upload.name}
                  </div>
                  <div className="text-xs text-muted-foreground truncate mb-1" title={upload.path}>
                    {upload.path}
                  </div>
                  <div className="flex items-center gap-2">
                    {upload.status === UploadStatus.uploading && (
                      <Progress 
                        value={upload.progress} 
                        className="flex-1 h-2"
                      />
                    )}
                    <div className="text-xs text-muted-foreground whitespace-nowrap">
                      {getStatusText(upload.status, upload.progress)}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
};
