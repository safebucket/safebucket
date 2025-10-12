import { useState } from "react";
import { UserPlus } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { EMAIL_REGEX, bucketGroups } from "@/types/bucket.ts";

export interface IAddMembersInputProps {
  onAddEmail: (email: string, group: string) => void;
}

export const AddMembersInput: FC<IAddMembersInputProps> = ({
  onAddEmail,
}: IAddMembersInputProps) => {
  const { t } = useTranslation();
  const [email, setEmail] = useState<string>("");
  const [group, setGroup] = useState<string>("viewer");

  const handleAddEmail = (e: React.MouseEvent) => {
    e.preventDefault();
    onAddEmail(email, group);
    setEmail("");
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      onAddEmail(email, group);
      setEmail("");
    }
  };

  return (
    <div className="space-y-4">
      <div className="text-sm font-medium">
        {t("bucket.settings.members.share_with")}
      </div>
      <div className="flex gap-3">
        <ButtonGroup className="w-full">
          <Input
            type="email"
            placeholder={t("bucket.settings.members.enter_email")}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onKeyDown={handleKeyPress}
            className="flex-1 w-full"
          />
          <Select value={group} onValueChange={setGroup}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="min-w-24">
              {bucketGroups.map((group) => (
                <SelectItem key={group.id} value={group.id}>
                  {group.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </ButtonGroup>
        <ButtonGroup>
          <Button
            aria-label={t("bucket.settings.members.add")}
            onClick={handleAddEmail}
            disabled={!email.trim() || !EMAIL_REGEX.test(email)}
            variant="outline"
            size="icon"
          >
            <UserPlus />
          </Button>
        </ButtonGroup>
      </div>
    </div>
  );
};
