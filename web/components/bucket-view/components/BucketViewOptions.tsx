import * as React from "react";
import { FC } from "react";

import { LayoutGrid, LayoutList } from "lucide-react";

import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { ButtonGroup } from "@/components/common/components/ButtonGroup";

interface IBucketViewOptionsProps {
  currentView: string;
  setCurrentView: (view: BucketViewMode) => void;
}

const options = [
  {
    key: BucketViewMode.List,
    value: <LayoutList />,
  },
  {
    key: BucketViewMode.Grid,
    value: <LayoutGrid />,
  },
];

export const BucketViewOptions: FC<IBucketViewOptionsProps> = ({
  currentView,
  setCurrentView,
}: IBucketViewOptionsProps) => {
  return (
    <ButtonGroup
      options={options}
      currentOption={currentView}
      setCurrentOption={setCurrentView}
    />
  );
};
