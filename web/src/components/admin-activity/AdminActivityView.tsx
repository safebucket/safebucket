import { useMemo } from "react";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAdminActivityData } from "./hooks/useAdminActivityData";
import { createColumns } from "./components/columns";
import { AdminActivityTable } from "./components/AdminActivityTable";
import type { FC } from "react";
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

  const { activities, isLoading, isFetching, refetch } = useAdminActivityData();

  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <Skeleton className="mb-6 h-8 w-48" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>{t("admin.activity.title")}</CardTitle>
            <CardDescription>{t("admin.activity.description")}</CardDescription>
          </div>
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
        </CardHeader>
        <CardContent>
          <AdminActivityTable columns={columns} data={activities} />
        </CardContent>
      </Card>
    </div>
  );
};
