import { useEffect, useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { IBucket } from "@/components/bucket-view/helpers/types";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";
import { api } from "@/lib/api.ts";

export interface IBucketInformationData {
  isEditingName: boolean;
  setIsEditingName: (editing: boolean) => void;
  bucketName: string;
  setBucketName: (name: string) => void;
  copiedField: string | null;
  bucketUrl: string;
  handleCopy: (text: string, field: string) => void;
  handleSaveName: () => void;
  handleCancelName: () => void;
}

export const useBucketInformation = (
  bucket: IBucket,
): IBucketInformationData => {
  const [isEditingName, setIsEditingName] = useState(false);
  const [bucketName, setBucketName] = useState(bucket.name);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const queryClient = useQueryClient();

  useEffect(() => {
    setBucketName(bucket.name);
  }, [bucket.name]);

  const updateNameMutation = useMutation({
    mutationFn: () => api.patch(`/buckets/${bucket.id}`, { name: bucketName }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      successToast("Bucket name updated successfully");
      setIsEditingName(false);
    },
    onError: (error: Error) => errorToast(error),
  });

  const bucketUrl = `${window.location.origin}/buckets/${bucket.id}`;

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedField(field);
      successToast(`${field} has been copied.`);
      setTimeout(() => setCopiedField(null), 2000);
    });
  };

  const handleSaveName = () => {
    if (!bucketName.trim()) {
      toast({
        variant: "destructive",
        title: "Invalid name",
        description: "Bucket name cannot be empty",
      });
      return;
    }

    if (bucketName === bucket.name) {
      setIsEditingName(false);
      return;
    }

    updateNameMutation.mutate();
  };

  const handleCancelName = () => {
    setBucketName(bucket.name);
    setIsEditingName(false);
  };

  return {
    isEditingName,
    setIsEditingName,
    bucketName,
    setBucketName,
    copiedField,
    bucketUrl,
    handleCopy,
    handleSaveName,
    handleCancelName,
  };
};
