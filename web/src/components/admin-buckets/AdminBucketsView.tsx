import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAdminBucketsData } from "./hooks/useAdminBucketsData";
import { createColumns } from "./components/columns";
import { AdminBucketsTable } from "./components/AdminBucketsTable";
import type { FC } from "react";
import type { IAdminBucket } from "@/queries/admin";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export const AdminBucketsView: FC = () => {
  const { t } = useTranslation();
  const deleteBucketDialog = useDialog();
  const columns = useMemo(() => createColumns(t), [t]);

  const {
    buckets,
    isLoading,
    deleteBucketMutation,
    bucketToDelete,
    setBucketToDelete,
  } = useAdminBucketsData();

  const handleDeleteClick = (bucket: IAdminBucket) => {
    setBucketToDelete(bucket);
    deleteBucketDialog.trigger();
  };

  const handleConfirmDelete = () => {
    if (bucketToDelete) {
      deleteBucketMutation.mutate(bucketToDelete.id);
      setBucketToDelete(null);
    }
  };

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
        <CardHeader>
          <CardTitle>{t("admin.buckets.title")}</CardTitle>
          <CardDescription>{t("admin.buckets.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <AdminBucketsTable
            columns={columns}
            data={buckets}
            onDeleteBucket={handleDeleteClick}
          />
        </CardContent>
      </Card>

      <CustomAlertDialog
        {...deleteBucketDialog.props}
        title={t("admin.buckets.delete_dialog.title")}
        description={t("admin.buckets.delete_dialog.description", {
          name: bucketToDelete?.name,
        })}
        cancelLabel={t("common.cancel")}
        confirmLabel={t("admin.buckets.delete_dialog.confirm")}
        onConfirm={handleConfirmDelete}
      />
    </div>
  );
};
