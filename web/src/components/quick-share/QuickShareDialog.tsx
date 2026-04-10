import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  ArrowLeft,
  ArrowRight,
  Link,
  Loader2,
  TriangleAlert,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import type { FC } from "react";

import type { BucketItem } from "@/types/bucket.ts";
import type { ShareScope } from "@/types/share.ts";
import { FileStatus } from "@/types/file.ts";
import { FolderStatus } from "@/types/folder.ts";
import { isFile } from "@/components/bucket-view/helpers/utils";

import { QuickShareOptionsStep } from "@/components/quick-share/components/QuickShareOptionsStep";
import { QuickShareResultStep } from "@/components/quick-share/components/QuickShareResultStep";
import { QuickShareScopeStep } from "@/components/quick-share/components/QuickShareScopeStep";
import { StepIndicator } from "@/components/quick-share/components/StepIndicator";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import {
  bucketDataQueryOptions,
  useCreateShareMutation,
} from "@/queries/bucket";

type Step = 1 | 2 | 3;

export interface IQuickShareForm {
  name: string;
  scope: ShareScope;
  selectedFileIds: Array<string>;
  selectedFolderId: string | null;
  hasExpiry: boolean;
  expiresAt: Date | undefined;
  limitViews: boolean;
  maxViews: number | "";
  passwordProtected: boolean;
  password: string;
  allowUploads: boolean;
  maxUploadSize: number | "";
  maxUploads: number | "";
}

interface IQuickShareDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialItem?: BucketItem;
  bucketId: string;
}

function getDefaultValues(
  t: (key: string) => string,
  initialItem?: BucketItem,
): IQuickShareForm {
  return {
    name: t("quick_share.default_name"),
    scope: initialItem ? (isFile(initialItem) ? "files" : "folder") : "bucket",
    selectedFileIds: initialItem && isFile(initialItem) ? [initialItem.id] : [],
    selectedFolderId:
      initialItem && !isFile(initialItem) ? initialItem.id : null,
    hasExpiry: false,
    expiresAt: undefined,
    limitViews: false,
    maxViews: 1,
    passwordProtected: false,
    password: "",
    allowUploads: false,
    maxUploadSize: 100,
    maxUploads: 1,
  };
}

export const QuickShareDialog: FC<IQuickShareDialogProps> = ({
  open,
  onOpenChange,
  initialItem,
  bucketId,
}) => {
  const { t } = useTranslation();
  const { data: bucket } = useQuery(bucketDataQueryOptions(bucketId));

  const { control, watch, setValue, reset, getValues } =
    useForm<IQuickShareForm>({
      defaultValues: getDefaultValues(t, initialItem),
    });

  const createShareMutation = useCreateShareMutation(bucketId);

  const scope = watch("scope");
  const selectedFileIds = watch("selectedFileIds");
  const selectedFolderId = watch("selectedFolderId");
  const hasExpiry = watch("hasExpiry");
  const limitViews = watch("limitViews");
  const passwordProtected = watch("passwordProtected");
  const allowUploads = watch("allowUploads");

  const [step, setStep] = useState<Step>(1);
  const [generatedLink, setGeneratedLink] = useState("");

  const allFolders = (bucket?.folders ?? []).filter(
    (f) => f.status === FolderStatus.created,
  );

  useEffect(() => {
    if (open) {
      reset(getDefaultValues(t, initialItem));
      setStep(1);
      setGeneratedLink("");
    }
  }, [open, initialItem, reset]);

  const handleScopeChange = (newScope: ShareScope) => {
    setValue("scope", newScope);
    setValue("allowUploads", false);
    if (newScope === "files") {
      setValue(
        "selectedFileIds",
        initialItem && isFile(initialItem) ? [initialItem.id] : [],
      );
      setValue("selectedFolderId", null);
    } else if (newScope === "folder") {
      setValue("selectedFileIds", []);
      setValue(
        "selectedFolderId",
        initialItem && !isFile(initialItem)
          ? initialItem.id
          : (allFolders[0]?.id ?? null),
      );
    } else {
      setValue("selectedFileIds", []);
      setValue("selectedFolderId", null);
    }
  };

  const handleToggleFile = (fileId: string) => {
    const current = selectedFileIds;
    setValue(
      "selectedFileIds",
      current.includes(fileId)
        ? current.filter((id) => id !== fileId)
        : [...current, fileId],
    );
  };

  const handleSelectFolder = (folderId: string) => {
    setValue("selectedFolderId", folderId);
  };

  const handleCreate = async () => {
    const values = getValues();

    const share = await createShareMutation.mutateAsync({
      name: values.name,
      type: values.scope,
      file_ids: values.scope === "files" ? values.selectedFileIds : undefined,
      folder_id:
        values.scope === "folder" ? values.selectedFolderId : undefined,
      expires_at:
        values.hasExpiry && values.expiresAt
          ? values.expiresAt.toISOString()
          : undefined,
      max_views:
        values.limitViews && values.maxViews
          ? Number(values.maxViews)
          : undefined,
      password:
        values.passwordProtected && values.password
          ? values.password
          : undefined,
      allow_upload: values.allowUploads,
      max_uploads:
        values.allowUploads && values.maxUploads
          ? Number(values.maxUploads)
          : undefined,
      max_upload_size:
        values.allowUploads && values.maxUploadSize
          ? Number(values.maxUploadSize) * 1024 * 1024
          : undefined,
    });

    setGeneratedLink(`${window.location.origin}/shares/${share.id}`);
    setStep(3);
  };

  const canProceedFromScope =
    scope === "bucket" ||
    (scope === "files" && selectedFileIds.length > 0) ||
    (scope === "folder" && selectedFolderId !== null);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        style={{ maxWidth: "640px" }}
        className="flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0"
      >
        <DialogHeader className="p-6 pb-4">
          <DialogTitle className="text-xl">
            {t("quick_share.title")}
          </DialogTitle>
          <DialogDescription className="sr-only" />
        </DialogHeader>

        <div className="px-6 pb-4">
          <StepIndicator current={step} />
        </div>

        <Separator />

        <div className="min-h-0 overflow-y-auto p-6">
          {step === 1 && (
            <QuickShareScopeStep
              scope={scope}
              selectedFileIds={selectedFileIds}
              selectedFolderId={selectedFolderId}
              files={(bucket?.files ?? []).filter(
                (f) => f.status === FileStatus.uploaded,
              )}
              folders={allFolders}
              onScopeChange={handleScopeChange}
              onToggleFile={handleToggleFile}
              onSelectFolder={handleSelectFolder}
            />
          )}

          {step === 2 && (
            <QuickShareOptionsStep
              scope={scope}
              control={control}
              hasExpiry={hasExpiry}
              limitViews={limitViews}
              passwordProtected={passwordProtected}
              allowUploads={allowUploads}
            />
          )}

          {step === 3 && <QuickShareResultStep generatedLink={generatedLink} />}
        </div>

        <Separator />

        <DialogFooter className="p-6 pt-4">
          {step === 1 && (
            <>
              <Button
                variant="outline"
                type="button"
                onClick={() => onOpenChange(false)}
              >
                {t("common.cancel")}
              </Button>
              <Button
                type="button"
                disabled={!canProceedFromScope}
                onClick={() => setStep(2)}
                className="gap-2"
              >
                {t("quick_share.next")}
                <ArrowRight className="h-4 w-4" />
              </Button>
            </>
          )}
          {step === 2 && (
            <div className="flex w-full flex-col gap-2">
              <div className="flex items-center gap-2 rounded-md border border-orange-200 bg-orange-50 px-3 py-2 text-sm text-orange-700 dark:border-orange-800 dark:bg-orange-950 dark:text-orange-400">
                <TriangleAlert className="h-4 w-4 shrink-0" />
                <span>{t("quick_share.public_link_warning")}</span>
              </div>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  type="button"
                  onClick={() => setStep(1)}
                  className="gap-2"
                >
                  <ArrowLeft className="h-4 w-4" />
                  {t("quick_share.back")}
                </Button>
                <Button
                  type="button"
                  onClick={handleCreate}
                  disabled={createShareMutation.isPending}
                  className="gap-2"
                >
                  {createShareMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Link className="h-4 w-4" />
                  )}
                  {t("quick_share.create")}
                </Button>
              </div>
            </div>
          )}
          {step === 3 && (
            <Button type="button" onClick={() => onOpenChange(false)}>
              {t("quick_share.done")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
