import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

interface UpdateUserPayload {
  first_name?: string;
  last_name?: string;
  old_password?: string;
  new_password?: string;
}

export const useUpdateUserMutation = (userId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateUserPayload) =>
      api.patch(`/users/${userId}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      successToast("Profile updated successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};
