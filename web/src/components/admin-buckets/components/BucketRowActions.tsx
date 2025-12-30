import { Ellipsis, Eye, Trash2 } from "lucide-react";
import { Link } from "@tanstack/react-router";
import type { FC } from "react";
import type { IAdminBucket } from "@/types/admin.ts";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface BucketRowActionsProps {
  bucket: IAdminBucket;
  onDelete: (bucket: IAdminBucket) => void;
}

export const BucketRowActions: FC<BucketRowActionsProps> = ({
  bucket,
  onDelete,
}) => {
  return (
    <div onClick={(e) => e.stopPropagation()}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="data-[state=open]:bg-muted flex h-8 w-8 p-0"
          >
            <Ellipsis className="h-4 w-4" />
            <span className="sr-only">Open bucket actions</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem asChild>
            <Link
              to="/buckets/$bucketId/{-$folderId}"
              params={{ bucketId: bucket.id, folderId: undefined }}
            >
              <Eye className="mr-2 h-4 w-4" />
              View
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => onDelete(bucket)}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};
