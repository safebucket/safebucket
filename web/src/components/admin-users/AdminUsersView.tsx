import { useMemo } from "react";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAdminUsersData } from "./hooks/useAdminUsersData";
import { createColumns } from "./components/columns";
import { AdminUsersTable } from "./components/AdminUsersTable";
import type { FC } from "react";
import type { FieldValues } from "react-hook-form";
import type { IUser } from "@/components/auth-view/types/session";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { useSession } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export const AdminUsersView: FC = () => {
  const { t } = useTranslation();
  const session = useSession();
  const createUserDialog = useDialog();
  const deleteUserDialog = useDialog();
  const columns = useMemo(() => createColumns(t), [t]);

  const {
    users,
    isLoading,
    createUserMutation,
    deleteUserMutation,
    userToDelete,
    setUserToDelete,
  } = useAdminUsersData();

  const handleCreateUser = (data: FieldValues) => {
    createUserMutation.mutate({
      first_name: data.first_name as string,
      last_name: data.last_name as string,
      email: data.email as string,
      password: data.password as string,
    });
  };

  const handleDeleteClick = (user: IUser) => {
    setUserToDelete(user);
    deleteUserDialog.trigger();
  };

  const handleConfirmDelete = () => {
    if (userToDelete) {
      deleteUserMutation.mutate(userToDelete.id);
      setUserToDelete(null);
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
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>{t("admin.users.title")}</CardTitle>
            <CardDescription>{t("admin.users.description")}</CardDescription>
          </div>
          <Button type="button" onClick={createUserDialog.trigger}>
            <Plus className="mr-2 h-4 w-4" />
            {t("admin.users.add_user")}
          </Button>
        </CardHeader>
        <CardContent>
          <AdminUsersTable
            columns={columns}
            data={users}
            onDeleteUser={handleDeleteClick}
            currentUserId={session?.userId ?? ""}
          />
        </CardContent>
      </Card>

      <FormDialog
        {...createUserDialog.props}
        maxWidth="650px"
        title={t("admin.users.add_user_dialog.title")}
        description={t("admin.users.add_user_dialog.description")}
        fields={[
          {
            id: "first_name",
            label: t("admin.users.add_user_dialog.first_name"),
            type: "text",
            required: true,
          },
          {
            id: "last_name",
            label: t("admin.users.add_user_dialog.last_name"),
            type: "text",
            required: true,
          },
          {
            id: "email",
            label: t("admin.users.add_user_dialog.email"),
            type: "text",
            required: true,
          },
          {
            id: "password",
            label: t("admin.users.add_user_dialog.password"),
            type: "password",
            required: true,
          },
        ]}
        onSubmit={handleCreateUser}
        confirmLabel={t("common.create")}
      />

      <CustomAlertDialog
        {...deleteUserDialog.props}
        title={t("admin.users.delete_dialog.title")}
        description={t("admin.users.delete_dialog.description", {
          email: userToDelete?.email,
        })}
        cancelLabel={t("common.cancel")}
        confirmLabel={t("admin.users.delete_dialog.confirm")}
        onConfirm={handleConfirmDelete}
      />
    </div>
  );
};
