import {
  Check,
  FolderClock,
  LayoutGrid,
  LayoutList,
  Settings,
  Trash2,
} from "lucide-react";
import { t } from "i18next";
import { useMemo } from "react";
import type { FC } from "react";

import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { ButtonGroup } from "@/components/ui/button-group.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip.tsx";

const options = [
  {
    key: BucketViewMode.List,
    value: <LayoutList />,
    tooltip: t("bucket.header.list_view"),
  },
  {
    key: BucketViewMode.Grid,
    value: <LayoutGrid />,
    tooltip: t("bucket.header.grid_view"),
  },
  {
    key: BucketViewMode.Activity,
    value: <FolderClock />,
    tooltip: t("bucket.header.activity"),
  },
  {
    key: BucketViewMode.Trash,
    value: <Trash2 />,
    tooltip: t("bucket.header.trash"),
  },
  {
    key: BucketViewMode.Settings,
    value: <Settings />,
    tooltip: t("bucket.header.settings"),
  },
];

interface IBucketViewOptionsProps {
  variant?: "buttons" | "dropdown";
}

export const BucketViewOptions: FC<IBucketViewOptionsProps> = ({
  variant = "buttons",
}) => {
  const { view, setView, bucketId } = useBucketViewContext();
  const { isOwner } = useBucketPermissions(bucketId);

  const filteredOptions = useMemo(() => {
    return options.filter(
      (opt) => !(opt.key === BucketViewMode.Settings && !isOwner),
    );
  }, [isOwner]);

  const activeOption = filteredOptions.find((opt) => opt.key === view);

  if (variant === "dropdown") {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="secondary">{activeOption?.value}</Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuGroup>
            {filteredOptions.map((opt, i) => (
              <DropdownMenuItem key={i} onClick={() => setView(opt.key)}>
                {opt.value}
                {opt.tooltip}
                {view === opt.key && <Check className="ml-auto h-4 w-4" />}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }

  return (
    <ButtonGroup className="default">
      {filteredOptions.map((opt, i) => (
        <TooltipProvider key={i}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant={view == opt.key ? "default" : "secondary"}
                onClick={() => setView(opt.key)}
              >
                {opt.value}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>{opt.tooltip}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ))}
    </ButtonGroup>
  );
};
