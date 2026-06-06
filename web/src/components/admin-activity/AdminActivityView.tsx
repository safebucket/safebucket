import { useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAdminActivityData } from "./hooks/useAdminActivityData";
import { createColumns } from "./components/columns";
import { AdminActivityTable } from "./components/AdminActivityTable";
import { ActivityFilters } from "./components/ActivityFilters";
import type { DateRange } from "react-day-picker";
import type { FC } from "react";
import type { ActivityMessage } from "@/types/activity";
import {
  ActivityDateRangePicker,
  dateRangeToQuery,
} from "@/components/activity-view/components/ActivityDateRangePicker";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export const AdminActivityView: FC = () => {
  const { t } = useTranslation();
  const columns = useMemo(() => createColumns(t), [t]);

  const [selectedActions, setSelectedActions] = useState<
    Array<ActivityMessage>
  >([]);
  const [selectedTypes, setSelectedTypes] = useState<Array<string>>([]);
  const [dateRange, setDateRange] = useState<DateRange | undefined>();

  const {
    activities,
    isLoading,
    isFetching,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useAdminActivityData({
    action: selectedActions,
    type: selectedTypes,
    ...dateRangeToQuery(dateRange),
  });

  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <Skeleton className="mb-6 h-8 w-48" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    );
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="container mx-auto p-6">
        <Card>
          <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <CardTitle>{t("admin.activity.title")}</CardTitle>
              <CardDescription>
                {t("admin.activity.description")}
              </CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <ActivityFilters
                selectedActions={selectedActions}
                selectedTypes={selectedTypes}
                onActionsChange={setSelectedActions}
                onTypesChange={setSelectedTypes}
              />
              <ActivityDateRangePicker
                value={dateRange}
                onChange={setDateRange}
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={() => refetch()}
                disabled={isFetching}
              >
                <RefreshCw
                  className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
                />
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <AdminActivityTable columns={columns} data={activities} />
            {hasNextPage && (
              <div className="flex justify-center">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage
                    ? t("activity.loading_more")
                    : t("activity.load_more")}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
