import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type { IFileComment } from "@/types/comment";
import { api } from "@/lib/api";
import { errorToast } from "@/lib/toast";

export const fileCommentsQueryOptions = (bucketId: string, fileId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "files", fileId, "comments"],
    queryFn: () =>
      api.get<{ data: Array<IFileComment> }>(
        `/buckets/${bucketId}/files/${fileId}/comments`,
      ),
    select: (response) => response.data,
  });

export const useFileCommentMutations = (bucketId: string, fileId: string) => {
  const queryClient = useQueryClient();
  const url = `/buckets/${bucketId}/files/${fileId}/comments`;
  const callbacks = {
    onError: (error: Error) => errorToast(error),
    onSettled: () =>
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "files", fileId, "comments"],
      }),
  };

  const create = useMutation({
    mutationFn: (body: string) => api.post<IFileComment>(url, { body }),
    ...callbacks,
  });

  const update = useMutation({
    mutationFn: ({ commentId, body }: { commentId: string; body: string }) =>
      api.patch(`${url}/${commentId}`, { body }),
    ...callbacks,
  });

  const remove = useMutation({
    mutationFn: (commentId: string) => api.delete(`${url}/${commentId}`),
    ...callbacks,
  });

  return { create, update, remove };
};
