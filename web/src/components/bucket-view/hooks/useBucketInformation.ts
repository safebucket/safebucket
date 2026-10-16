import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { IBucket } from "@/types/bucket.ts";
import { api } from "@/lib/api.ts";
import { errorToast } from "@/lib/toast";

export interface IBucketInformationData {
  isEditingName: boolean;
  setIsEditingName: (editing: boolean) => void;
  bucketName: string;
  setBucketName: (name: string) => void;
  isEditingDescription: boolean;
  setIsEditingDescription: (editing: boolean) => void;
  bucketDescription: string;
  setBucketDescription: (description: string) => void;
  copiedField: string | null;
  bucketUrl: string;
  handleCopy: (text: string, field: string) => void;
  handleSaveName: () => void;
  handleCancelName: () => void;
  handleSaveDescription: () => void;
  handleCancelDescription: () => void;
  isUpdating: boolean;
}

interface IUpdateBucketPayload {
  name: string;
  description?: string;
}

export const useBucketInformation = (
  bucket: IBucket,
): IBucketInformationData => {
  const { t } = useTranslation();
  const [isEditingName, setIsEditingName] = useState(false);
  const [bucketName, setBucketName] = useState(bucket.name);
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [bucketDescription, setBucketDescription] = useState(
    bucket.description,
  );
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const queryClient = useQueryClient();

  useEffect(() => {
    setBucketName(bucket.name);
  }, [bucket.name]);

  useEffect(() => {
    setBucketDescription(bucket.description);
  }, [bucket.description]);

  const updateBucketMutation = useMutation({
    mutationFn: ({ name, description }: IUpdateBucketPayload) =>
      api.patch(`/buckets/${bucket.id}`, { name, description }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["buckets"] }),
  });

  const bucketUrl = `${window.location.origin}/buckets/${bucket.id}`;

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedField(field);
      toast.success(t("toast.copied", { field }));
      setTimeout(() => setCopiedField(null), 2000);
    });
  };

  const handleSaveName = () => {
    if (!bucketName.trim()) {
      errorToast(t("toast.invalid_name"), t("toast.bucket_name_empty"));
      return;
    }

    if (bucketName === bucket.name) {
      setIsEditingName(false);
      return;
    }

    updateBucketMutation.mutate(
      { name: bucketName },
      {
        onSuccess: () => {
          toast.success(t("toast.bucket_name_updated"));
          setIsEditingName(false);
        },
      },
    );
  };

  const handleCancelName = () => {
    setBucketName(bucket.name);
    setIsEditingName(false);
  };

  const handleSaveDescription = () => {
    const trimmedDescription = bucketDescription.trim();

    if (trimmedDescription === bucket.description) {
      setBucketDescription(bucket.description);
      setIsEditingDescription(false);
      return;
    }

    updateBucketMutation.mutate(
      { name: bucket.name, description: trimmedDescription },
      {
        onSuccess: () => {
          toast.success(t("toast.bucket_description_updated"));
          setIsEditingDescription(false);
        },
      },
    );
  };

  const handleCancelDescription = () => {
    setBucketDescription(bucket.description);
    setIsEditingDescription(false);
  };

  return {
    isEditingName,
    setIsEditingName,
    bucketName,
    setBucketName,
    isEditingDescription,
    setIsEditingDescription,
    bucketDescription,
    setBucketDescription,
    copiedField,
    bucketUrl,
    handleCopy,
    handleSaveName,
    handleCancelName,
    handleSaveDescription,
    handleCancelDescription,
    isUpdating: updateBucketMutation.isPending,
  };
};
