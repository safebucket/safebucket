import { ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { formatAction } from "../helpers/format";
import type { FC } from "react";
import { ActivityMessage } from "@/types/activity";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const RESOURCE_TYPES = [
  "bucket",
  "file",
  "folder",
  "user",
  "mfa_device",
  "share",
] as const;

const ACTIONS = Object.values(ActivityMessage);

interface ActivityFiltersProps {
  selectedActions: Array<ActivityMessage>;
  selectedTypes: Array<string>;
  onActionsChange: (value: Array<ActivityMessage>) => void;
  onTypesChange: (value: Array<string>) => void;
}

const toggle = <T,>(list: Array<T>, value: T): Array<T> =>
  list.includes(value) ? list.filter((v) => v !== value) : [...list, value];

export const ActivityFilters: FC<ActivityFiltersProps> = ({
  selectedActions,
  selectedTypes,
  onActionsChange,
  onTypesChange,
}) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            {t("admin.activity.filters.action")}
            {selectedActions.length > 0 && (
              <Badge variant="secondary" className="ml-1 px-1.5">
                {selectedActions.length}
              </Badge>
            )}
            <ChevronDown className="size-4 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="max-h-80 overflow-y-auto">
          {ACTIONS.map((action) => (
            <DropdownMenuCheckboxItem
              key={action}
              checked={selectedActions.includes(action)}
              onCheckedChange={() =>
                onActionsChange(toggle(selectedActions, action))
              }
              onSelect={(e) => e.preventDefault()}
            >
              {formatAction(action)}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            {t("admin.activity.filters.type")}
            {selectedTypes.length > 0 && (
              <Badge variant="secondary" className="ml-1 px-1.5">
                {selectedTypes.length}
              </Badge>
            )}
            <ChevronDown className="size-4 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          {RESOURCE_TYPES.map((type) => (
            <DropdownMenuCheckboxItem
              key={type}
              checked={selectedTypes.includes(type)}
              onCheckedChange={() => onTypesChange(toggle(selectedTypes, type))}
              onSelect={(e) => e.preventDefault()}
            >
              {t(`admin.activity.types.${type}`)}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};
