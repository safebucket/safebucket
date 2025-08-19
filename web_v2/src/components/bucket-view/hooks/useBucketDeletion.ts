import { useState } from "react";

import { useNavigate } from "@tanstack/react-router";
import { mutate } from "swr";

import type { IBucket } from "@/components/bucket-view/helpers/types";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";
import { api_deleteBucket } from "@/components/upload/helpers/api";

export interface IBucketDeletionData {
  confirmationText: string;
  setConfirmationText: (text: string) => void;
  expectedDeleteText: string;
  isConfirmationValid: boolean;
  handleDeleteBucket: () => Promise<void>;
}

export const useBucketDeletion = (bucket: IBucket): IBucketDeletionData => {
  const navigate = useNavigate();

  const [confirmationText, setConfirmationText] = useState("");

  const expectedDeleteText = `delete ${bucket.name}`;
  const isConfirmationValid = confirmationText === expectedDeleteText;

  const handleDeleteBucket = async () => {
    if (!isConfirmationValid) {
      toast({
        variant: "destructive",
        title: "Invalid confirmation",
        description: `Please type "${expectedDeleteText}" to confirm deletion`,
      });
      return;
    }

    api_deleteBucket(bucket.id)
      .then(() => {
        mutate("/buckets").then(() => {
          navigate({ to: "/" });
          successToast(`Bucket ${bucket.name} has been deleted`);
        });
      })
      .catch(errorToast);
  };

  return {
    confirmationText,
    setConfirmationText,
    expectedDeleteText,
    isConfirmationValid,
    handleDeleteBucket,
  };
};
