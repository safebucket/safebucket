import { useState } from "react";

import { useNavigate } from "@tanstack/react-router";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { IBucket } from "@/types/bucket.ts";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";
import { api } from "@/lib/api.ts";

export interface IBucketDeletionData {
  confirmationText: string;
  setConfirmationText: (text: string) => void;
  expectedDeleteText: string;
  isConfirmationValid: boolean;
  handleDeleteBucket: () => void;
}

export const useBucketDeletion = (bucket: IBucket): IBucketDeletionData => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [confirmationText, setConfirmationText] = useState("");

  const deleteMutation = useMutation({
    mutationFn: () => api.delete(`/buckets/${bucket.id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      navigate({ to: "/" });
      successToast(`Bucket ${bucket.name} has been deleted`);
    },
    onError: (error: Error) => errorToast(error),
  });

  const expectedDeleteText = `delete ${bucket.name}`;
  const isConfirmationValid = confirmationText === expectedDeleteText;

  const handleDeleteBucket = () => {
    if (!isConfirmationValid) {
      toast({
        variant: "destructive",
        title: "Invalid confirmation",
        description: `Please type "${expectedDeleteText}" to confirm deletion`,
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
