import React, { FC } from "react";

import { ChevronDownIcon, CircleCheck, FileIcon, Upload, AlertCircle, Clock } from "lucide-react";

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

const getStatusIcon = (status: UploadStatus, progress: number) => {
  switch (status) {
    case UploadStatus.success:
      return <CircleCheck className="h-5 w-5 text-green-500" />;
    case UploadStatus.failed:
      return <AlertCircle className="h-5 w-5 text-red-500" />;
    case UploadStatus.uploading:
      if (progress === 0) {
        return <Clock className="h-5 w-5 text-blue-500" />;
      }
      return <Upload className="h-5 w-5 text-blue-500" />;
    default:
      return <FileIcon className="h-5 w-5 text-muted-foreground" />;
  }
};

const getStatusText = (status: UploadStatus, progress: number) => {
  switch (status) {
    case UploadStatus.success:
      return "Completed";
    case UploadStatus.failed:
      return "Failed";
    case UploadStatus.uploading:
      if (progress === 0) {
        return "Preparing...";
      }
      return `${progress}%`;
    default:
      return "Unknown";
  }
};

const getProgressColor = (status: UploadStatus) => {
  switch (status) {
    case UploadStatus.success:
      return "bg-green-500";
    case UploadStatus.failed:
      return "bg-red-500";
    case UploadStatus.uploading:
      return "bg-blue-500";
    default:
      return "bg-gray-500";
  }
};

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
            {/* Summary header */}
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
            
            {/* Upload items */}
            {uploads.map((upload) => (
              <div key={upload.id} className="flex items-center gap-3 p-2 rounded hover:bg-muted/30">
                {/* Status icon */}
                <div className="flex-shrink-0">
                  {getStatusIcon(upload.status, upload.progress)}
                </div>
                
                {/* File info */}
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
