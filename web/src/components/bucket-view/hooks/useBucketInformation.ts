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
  copiedField: string | null;
  bucketUrl: string;
  handleCopy: (text: string, field: string) => void;
  handleSaveName: () => void;
  handleCancelName: () => void;
}

export const useBucketInformation = (
  bucket: IBucket,
): IBucketInformationData => {
  const { t } = useTranslation();
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
      toast.success(t("common.success"), {
        description: t("toast.bucket_name_updated"),
      });
      setIsEditingName(false);
    },
  });

  const bucketUrl = `${window.location.origin}/buckets/${bucket.id}`;

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedField(field);
      toast.success(t("common.success"), {
        description: t("toast.copied", { field }),
      });
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
