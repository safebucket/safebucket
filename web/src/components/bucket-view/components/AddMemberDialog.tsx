import { useTranslation } from "react-i18next";
import type { FC } from "react";
import { bucketGroups } from "@/types/bucket.ts";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface IAddMemberDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  email: string;
  onEmailChange: (email: string) => void;
  group: string;
  onGroupChange: (group: string) => void;
  canAdd: boolean;
  onAdd: () => void;
}

export const AddMemberDialog: FC<IAddMemberDialogProps> = ({
  open,
  onOpenChange,
  email,
  onEmailChange,
  group,
  onGroupChange,
  canAdd,
  onAdd,
}: IAddMemberDialogProps) => {
  const { t } = useTranslation();

  const handleAdd = () => {
    onAdd();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("bucket.settings.members.add_member")}</DialogTitle>
          <DialogDescription>
            {t("bucket.settings.members.description")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="add-member-email">
              {t("bucket.view.members.email_label")}
            </Label>
            <Input
              id="add-member-email"
              type="email"
              placeholder={t("bucket.settings.members.enter_email")}
              value={email}
              onChange={(e) => onEmailChange(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label>{t("bucket.view.members.role_label")}</Label>
            <Select value={group} onValueChange={onGroupChange}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {bucketGroups.map((g) => (
                  <SelectItem key={g.id} value={g.id}>
                    {g.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleAdd} disabled={!canAdd}>
            {t("bucket.settings.members.add_member")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
