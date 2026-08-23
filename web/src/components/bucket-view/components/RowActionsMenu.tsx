import { EllipsisVertical } from "lucide-react";
import type { FC } from "react";
import type { BucketItem } from "@/types/bucket.ts";
import { FileActions } from "@/components/file-actions/FileActions";
import { Button } from "@/components/ui/button";

interface IRowActionsMenuProps {
  item: BucketItem;
}

export const RowActionsMenu: FC<IRowActionsMenuProps> = ({
  item,
}: IRowActionsMenuProps) => {
  return (
    <FileActions file={item} type="dropdown">
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={(e) => e.stopPropagation()}
      >
        <EllipsisVertical className="h-4 w-4" />
      </Button>
    </FileActions>
  );
};
