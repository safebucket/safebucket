import { useState } from "react";

import { mutate } from "swr";

import { IBucket } from "@/components/bucket-view/helpers/types";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";
import { api_updateBucketName } from "@/components/upload/helpers/api";

export interface IBucketInformationData {
  isEditingName: boolean;
  setIsEditingName: (editing: boolean) => void;
  bucketName: string;
  setBucketName: (name: string) => void;
  copiedField: string | null;
  bucketUrl: string;
  handleCopy: (text: string, field: string) => Promise<void>;
  handleSaveName: () => Promise<void>;
  handleCancelName: () => void;
}

export const useBucketInformation = (
  bucket: IBucket,
): IBucketInformationData => {
  const [isEditingName, setIsEditingName] = useState(false);
  const [bucketName, setBucketName] = useState(bucket.name);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const bucketUrl = `${window.location.origin}/buckets/${bucket.id}`;

  const handleCopy = async (text: string, field: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedField(field);
      successToast(`${field} has been copied.`);
      setTimeout(() => setCopiedField(null), 2000);
    });
  };

  const handleSaveName = async () => {
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

    api_updateBucketName(bucket.id, bucketName)
      .then(() =>
        mutate(`/buckets/${bucket.id}`).then(() => {
          successToast("Bucket name updated successfully");
          setIsEditingName(false);
        }),
      )
      .catch(errorToast);
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
