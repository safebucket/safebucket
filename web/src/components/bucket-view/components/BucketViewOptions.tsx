import {
  FolderClock,
  LayoutGrid,
  LayoutList,
  Settings,
  Trash2,
} from "lucide-react";
import { t } from "i18next";
import type { FC } from "react";

import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { ButtonGroup } from "@/components/ui/button-group.tsx";
import { Button } from "@/components/ui/button.tsx";
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

export const BucketViewOptions: FC = () => {
  const { view, setView } = useBucketViewContext();

  return (
    <ButtonGroup className="default">
      {options.map((opt, i) => (
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
