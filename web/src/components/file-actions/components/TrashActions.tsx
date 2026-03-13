import { useTranslation } from "react-i18next";

import { ArchiveRestore, Trash2 } from "lucide-react";
import type { FC, ReactNode } from "react";

import type { BucketItem } from "@/types/bucket.ts";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useBucketPermissions } from "@/hooks/usePermissions";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface ITrashActionsProps {
  children: ReactNode;
  file: BucketItem;
  type: "context" | "dropdown";
  onRestore?: (fileId: string, fileName: string) => void;
  onPermanentDelete?: (fileId: string, fileName: string) => void;
}

export const TrashActions: FC<ITrashActionsProps> = ({
  children,
  file,
  type,
  onRestore,
  onPermanentDelete,
}: ITrashActionsProps) => {
  const { t } = useTranslation();
  const { bucketId } = useBucketViewContext();
  const { isContributor } = useBucketPermissions(bucketId);

  const Menu = type === "context" ? ContextMenu : DropdownMenu;
  const MenuTrigger =
    type === "context" ? ContextMenuTrigger : DropdownMenuTrigger;
  const MenuContent =
    type === "context" ? ContextMenuContent : DropdownMenuContent;
  const MenuItem = type === "context" ? ContextMenuItem : DropdownMenuItem;
  const Separator =
    type === "context" ? ContextMenuSeparator : DropdownMenuSeparator;

  return (
    <Menu>
      <MenuTrigger asChild>{children}</MenuTrigger>
      <MenuContent className="w-45">
        {isContributor && (
          <>
            <MenuItem onClick={() => onRestore?.(file.id, file.name)}>
              <ArchiveRestore className="mr-2 h-4 w-4" />
              {t("bucket.trash_view.restore")}
            </MenuItem>
            <Separator />
            <MenuItem
              className="text-red-600"
              onClick={() => onPermanentDelete?.(file.id, file.name)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              {t("bucket.trash_view.delete_permanently")}
            </MenuItem>
          </>
        )}
      </MenuContent>
    </Menu>
  );
};
