import { Upload } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type React from "react";
import type { DragEvent, FC } from "react";
import type { FileWithPath } from "@/components/upload/helpers/types";
import { useToast } from "@/components/ui/hooks/use-toast.ts";
import { extractFilesFromDrop } from "@/components/upload/helpers/file-processing";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { cn } from "@/lib/utils";

interface IDragDropZoneProps {
  bucketId: string;
  children: React.ReactNode;
  className?: string;
  onFilesDropped: (files: Array<FileWithPath>) => void;
}

export const DragDropZone: FC<IDragDropZoneProps> = ({
  bucketId,
  children,
  className,
  onFilesDropped,
}) => {
  const { t } = useTranslation();
  const { toast } = useToast();
  const [isDragOver, setIsDragOver] = useState(false);
  const dragCounterRef = useRef(0);

  const { isContributor } = useBucketPermissions(bucketId);

  const handleDragEnter = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    dragCounterRef.current += 1;

    if (e.dataTransfer.types.includes("Files")) {
      setIsDragOver(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    dragCounterRef.current -= 1;

    if (dragCounterRef.current <= 0) {
      dragCounterRef.current = 0;
      setIsDragOver(false);
    }
  }, []);

  const handleDragOver = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    if (e.dataTransfer.types.includes("Files")) {
      e.dataTransfer.dropEffect = "copy";
    }
  }, []);

  const handleDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

      setIsDragOver(false);
      dragCounterRef.current = 0;

      if (!isContributor) {
        toast({
          variant: "destructive",
          description: t("upload.permission_denied"),
        });
        return;
      }

      const files = await extractFilesFromDrop(e.dataTransfer);
      if (files.length > 0) {
        onFilesDropped(files);
      }
    },
    [isContributor, toast, t, onFilesDropped],
  );

  return (
    <div
      className={cn("relative", className)}
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {children}

      {isDragOver && (
        <div className="bg-primary/10 border-primary fixed inset-0 z-50 flex items-center justify-center border-4 border-dashed">
          <div className="text-primary flex flex-col items-center justify-center space-y-4">
            <div className="relative">
              <Upload className="h-20 w-20" />
            </div>
            <div className="text-center">
              <p className="text-xl font-semibold">
                {t("upload.drag_drop.drop_files")}
              </p>
              <p className="text-primary/80 text-base">
                {t("upload.drag_drop.drop_description")}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
