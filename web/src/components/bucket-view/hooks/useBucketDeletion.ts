import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useNavigate } from "@tanstack/react-router";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { IBucket } from "@/types/bucket.ts";
import { errorToast, toast } from "@/components/ui/hooks/use-toast";
import { api } from "@/lib/api.ts";

export interface IBucketDeletionData {
  confirmationText: string;
  setConfirmationText: (text: string) => void;
  expectedDeleteText: string;
  isConfirmationValid: boolean;
  handleDeleteBucket: () => void;
}

export const useBucketDeletion = (bucket: IBucket): IBucketDeletionData => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [confirmationText, setConfirmationText] = useState("");

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/buckets/${bucket.id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      navigate({ to: "/" });
      toast({
        variant: "success",
        title: t("common.success"),
        description: t("toast.bucket_deleted", { name: bucket.name }),
      });
    },
    onError: (error: Error) => errorToast(error),
  });

  const expectedDeleteText = `delete ${bucket.name}`;
  const isConfirmationValid = confirmationText === expectedDeleteText;

  const handleDeleteBucket = () => {
    if (!isConfirmationValid) {
      toast({
        variant: "destructive",
        title: t("toast.invalid_confirmation"),
        description: t("toast.confirm_deletion_prompt", {
          text: expectedDeleteText,
        }),
      });
      return;
    }

    deleteMutation.mutate();
  };

  return {
    confirmationText,
    setConfirmationText,
    expectedDeleteText,
    isConfirmationValid,
    handleDeleteBucket,
  };
};
