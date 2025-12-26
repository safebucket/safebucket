import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { IUser } from "@/components/auth-view/types/session";
import {
  useCreateUserMutation,
  useDeleteUserMutation,
  usersQueryOptions,
} from "@/queries/admin";

export const useAdminUsersData = () => {
  const [userToDelete, setUserToDelete] = useState<IUser | null>(null);

  const { data: users, isLoading } = useQuery(usersQueryOptions());
  const createUserMutation = useCreateUserMutation();
  const deleteUserMutation = useDeleteUserMutation();

  return {
    users: users ?? [],
    isLoading,
    createUserMutation,
    deleteUserMutation,
    userToDelete,
    setUserToDelete,
  };
};
