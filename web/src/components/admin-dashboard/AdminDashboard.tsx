import { useState } from "react";
import { FileText, FolderOpen, HardDrive, Users } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAdminDashboardData } from "./hooks/useAdminDashboardData";
import { StatCard } from "./components/StatCard";
import { SharedFilesChart } from "./components/SharedFilesChart";
import type { FC } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { formatFileSize } from "@/lib/utils.ts";

export const AdminDashboard: FC = () => {
  const { t } = useTranslation();
  const [timeRange, setTimeRange] = useState("90");
  const { stats, isLoading } = useAdminDashboardData(Number(timeRange));

  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <Skeleton className="mb-6 h-8 w-48" />
        <div className="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Skeleton className="h-[120px]" />
          <Skeleton className="h-[120px]" />
          <Skeleton className="h-[120px]" />
          <Skeleton className="h-[120px]" />
        </div>
        <div className="mb-6 grid gap-4 md:grid-cols-2">
          <Skeleton className="h-[280px]" />
          <Skeleton className="h-[280px]" />
        </div>
        <Skeleton className="h-[400px]" />
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 pt-0">
      <div className="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title={t("admin.dashboard.stats.users")}
          value={stats?.total_users ?? 0}
          icon={Users}
          href="/admin/users"
        />
        <StatCard
          title={t("admin.dashboard.stats.buckets")}
          value={stats?.total_buckets ?? 0}
          icon={FolderOpen}
        />
        <StatCard
          title={t("admin.dashboard.stats.files")}
          value={stats?.total_files ?? 0}
          icon={FileText}
        />
        <StatCard
          title={t("admin.dashboard.stats.storage")}
          value={formatFileSize(stats?.total_storage ?? 0)}
          icon={HardDrive}
        />
      </div>

      <SharedFilesChart
        data={stats?.shared_files ?? []}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
      />
    </div>
  );
};
