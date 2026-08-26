import { Fragment } from "react";
import { ArrowLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { getFolderPathTrail } from "@/components/bucket-view/helpers/utils";
import { DroppableArea } from "@/components/bucket-view/components/DroppableArea";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "@/components/ui/button";

interface IFolderPathBarProps {
  bucket: IBucket;
  folderId: string | undefined;
}

export const FolderPathBar: FC<IFolderPathBarProps> = ({
  bucket,
  folderId,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const trail = getFolderPathTrail(bucket.folders, folderId);
  const parentIndex = trail.length - 2;
  const parentFolder = parentIndex >= 0 ? trail[parentIndex] : undefined;

  const navigateTo = (targetFolderId: string | undefined) => {
    navigate({
      to: "/buckets/$bucketId/files/{-$folderId}",
      params: { bucketId: bucket.id, folderId: targetFolderId },
    });
  };

  return (
    <div className="flex items-center gap-1">
      <DroppableArea
        folderId={parentFolder?.id}
        disabled={folderId === undefined}
      >
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          aria-label={t("bucket.view.go_back")}
          disabled={!folderId}
          onClick={() => navigateTo(parentFolder?.id)}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
      </DroppableArea>
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem
            className="cursor-pointer rounded px-1 py-0.5"
            onClick={() => navigateTo(undefined)}
          >
            {trail.length === 0 ? (
              <BreadcrumbPage>{bucket.name}</BreadcrumbPage>
            ) : (
              <DroppableArea
                folderId={undefined}
                disabled={folderId === undefined}
              >
                <span className="hover:text-foreground">{bucket.name}</span>
              </DroppableArea>
            )}
          </BreadcrumbItem>
          {trail.map((folder, index) => {
            const isLast = index === trail.length - 1;
            return (
              <Fragment key={folder.id}>
                <BreadcrumbSeparator />
                <BreadcrumbItem
                  className="cursor-pointer rounded px-1 py-0.5"
                  onClick={isLast ? undefined : () => navigateTo(folder.id)}
                >
                  {isLast ? (
                    <BreadcrumbPage>{folder.name}</BreadcrumbPage>
                  ) : (
                    <DroppableArea folderId={folder.id}>
                      <span className="hover:text-foreground">
                        {folder.name}
                      </span>
                    </DroppableArea>
                  )}
                </BreadcrumbItem>
              </Fragment>
            );
          })}
        </BreadcrumbList>
      </Breadcrumb>
    </div>
  );
};
