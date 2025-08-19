import * as React from "react";
import { FC } from "react";

import { FolderClock, LayoutGrid, LayoutList, Settings } from "lucide-react";

import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { ButtonGroup } from "@/components/common/components/ButtonGroup";

const options = [
  {
    key: BucketViewMode.List,
    value: <LayoutList />,
    tooltip: "List view",
  },
  {
    key: BucketViewMode.Grid,
    value: <LayoutGrid />,
    tooltip: "Grid view",
  },
  {
    key: BucketViewMode.Activity,
    value: <FolderClock />,
    tooltip: "Bucket activity",
  },
  {
    key: BucketViewMode.Settings,
    value: <Settings />,
    tooltip: "Bucket settings",
  },
];

export const BucketViewOptions: FC = () => {
  const { view, setView } = useBucketViewContext();

  return (
    <ButtonGroup
      options={options}
      currentOption={view}
      setCurrentOption={setView}
    />
  );
};
