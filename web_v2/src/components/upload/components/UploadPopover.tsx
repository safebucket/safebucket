import type { FC } from "react";

import { ChevronDownIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Progress } from "@/components/ui/progress";
import { UploadStatus } from "@/components/upload/helpers/types";
import {
  getStatusIcon,
  getStatusText,
} from "@/components/upload/helpers/utils";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

export const UploadPopover: FC = () => {
  const { uploads } = useUploadContext();

  const activeUploads = uploads.filter(
    (upload) => upload.status !== UploadStatus.success,
  );
  const completedCount = uploads.filter(
    (upload) => upload.status === UploadStatus.success,
  ).length;
  const failedCount = uploads.filter(
    (upload) => upload.status === UploadStatus.failed,
  ).length;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" className="relative">
          Uploads
          {uploads.length > 0 && (
            <Badge className="absolute -top-2 -right-2 h-6 w-6 justify-center">
              {uploads.length}
            </Badge>
          )}
          <ChevronDownIcon className="ml-2 h-4 w-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="max-h-80 w-96 overflow-y-auto">
        {!uploads.length && (
          <p className="text-muted-foreground flex items-center justify-center py-4">
            No uploads in progress
          </p>
        )}

        {uploads.length > 0 && (
          <div className="space-y-2">
            {(completedCount > 0 || failedCount > 0) && (
              <div className="bg-muted/50 flex items-center justify-between rounded p-2 text-xs">
                <span>
                  {completedCount > 0 && (
                    <span className="text-green-600">
                      {completedCount} completed
                    </span>
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
              <div
                key={upload.id}
                className="hover:bg-muted/30 flex items-center gap-3 rounded p-2"
              >
                <div className="flex-shrink-0">
                  {getStatusIcon(upload.status, upload.progress)}
                </div>

                <div className="min-w-0 flex-1">
                  <div
                    className="truncate text-sm font-medium"
                    title={upload.path}
                  >
                    {upload.name}
                  </div>
                  <div
                    className="text-muted-foreground mb-1 truncate text-xs"
                    title={upload.path}
                  >
                    {upload.path}
                  </div>
                  <div className="flex items-center gap-2">
                    {upload.status === UploadStatus.uploading && (
                      <Progress
                        value={upload.progress}
                        className="h-2 flex-1"
                      />
                    )}
                    <div className="text-muted-foreground text-xs whitespace-nowrap">
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
