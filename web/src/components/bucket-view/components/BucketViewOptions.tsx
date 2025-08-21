import { FolderClock, LayoutGrid, LayoutList, Settings } from "lucide-react";
import { t } from "i18next";
import type { FC } from "react";

import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { ButtonGroup } from "@/components/common/components/ButtonGroup";

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
    key: BucketViewMode.Settings,
    value: <Settings />,
    tooltip: t("bucket.header.settings"),
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
