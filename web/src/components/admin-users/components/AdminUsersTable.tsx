import { useState } from "react";
import { flexRender, useTable } from "@tanstack/react-table";
import { useTranslation } from "react-i18next";
import { UserRowActions } from "./UserRowActions";
import { features } from "./columns";
import type { ColumnDef, SortingState } from "@tanstack/react-table";
import type { AdminTableFeatures } from "./columns";
import type { IAdminUser } from "@/components/auth-view/types/session";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface AdminUsersTableProps {
  columns: Array<ColumnDef<AdminTableFeatures, IAdminUser>>;
  data: Array<IAdminUser>;
  onDeleteUser: (user: IAdminUser) => void;
  currentUserId: string;
}

export function AdminUsersTable({
  columns,
  data,
  onDeleteUser,
  currentUserId,
}: AdminUsersTableProps) {
  const { t } = useTranslation();
  const [sorting, setSorting] = useState<SortingState>([]);

  const table = useTable({
    data,
    columns,
    features,
    state: { sorting },
    onSortingChange: setSorting,
  });

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                </TableHead>
              ))}
              <TableHead className="w-[50px]">
                {t("admin.users.columns.actions")}
              </TableHead>
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {table.getRowModel().rows.length ? (
            table.getRowModel().rows.map((row) => (
              <TableRow key={row.id}>
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </TableCell>
                ))}
                <TableCell>
                  <UserRowActions
                    user={row.original}
                    onDelete={onDeleteUser}
                    isCurrentUser={row.original.id === currentUserId}
                  />
                </TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell
                colSpan={columns.length + 1}
                className="h-24 text-center"
              >
                {t("admin.users.no_users")}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
