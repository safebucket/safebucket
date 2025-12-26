import { formatDistanceToNow } from "date-fns";
import type { TFunction } from "i18next";
import type { ColumnDef } from "@tanstack/react-table";
import type { IUser } from "@/components/auth-view/types/session";
import { Badge } from "@/components/ui/badge";

export const createColumns = (t: TFunction): Array<ColumnDef<IUser>> => [
  {
    accessorKey: "email",
    header: t("admin.users.columns.email"),
  },
  {
    accessorKey: "first_name",
    header: t("admin.users.columns.first_name"),
  },
  {
    accessorKey: "last_name",
    header: t("admin.users.columns.last_name"),
  },
  {
    accessorKey: "role",
    header: t("admin.users.columns.role"),
    cell: ({ row }) => {
      const role: string = row.getValue("role");
      return (
        <Badge variant={role === "admin" ? "default" : "secondary"}>
          {role}
        </Badge>
      );
    },
  },
  {
    accessorKey: "provider_type",
    header: t("admin.users.columns.provider"),
    cell: ({ row }) => {
      const provider: string = row.getValue("provider_type");
      return provider.charAt(0).toUpperCase() + provider.slice(1);
    },
  },
  {
    accessorKey: "created_at",
    header: t("admin.users.columns.created"),
    cell: ({ row }) => {
      return formatDistanceToNow(new Date(row.getValue("created_at")), {
        addSuffix: true,
      });
    },
  },
];
