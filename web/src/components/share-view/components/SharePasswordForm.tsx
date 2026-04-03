import { useState } from "react";
import { ArrowLeft, Loader2, Lock } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { Button } from "@/components/ui/button.tsx";
import { Card } from "@/components/ui/card.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Label } from "@/components/ui/label.tsx";
import { FormErrorAlert } from "@/components/common/FormErrorAlert.tsx";

interface ISharePasswordFormProps {
  onSubmit: (password: string) => void;
  onBack: () => void;
  isLoading: boolean;
  error: string | null;
}

export const SharePasswordForm: FC<ISharePasswordFormProps> = ({
  onSubmit,
  onBack,
  isLoading,
  error,
}) => {
  const { t } = useTranslation();
  const [password, setPassword] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (password.trim()) {
      onSubmit(password);
    }
  };

  return (
    <div className="flex min-h-svh items-center justify-center p-6">
      <Card className="flex w-full max-w-md flex-col gap-6 p-8">
        <div className="flex flex-col items-center gap-4">
          <div className="bg-primary/10 flex h-16 w-16 items-center justify-center rounded-full">
            <Lock className="text-primary h-8 w-8" />
          </div>
          <div className="space-y-2 text-center">
            <h1 className="text-xl font-semibold">
              {t("share_consumer.password_required")}
            </h1>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="space-y-2">
            <Label htmlFor="share-password">{t("auth.password")}</Label>
            <Input
              id="share-password"
              type="password"
              placeholder={t("share_consumer.password_placeholder")}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <FormErrorAlert error={error} />

          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={onBack}
              className="gap-2"
            >
              <ArrowLeft className="h-4 w-4" />
              {t("share_consumer.back")}
            </Button>
            <Button
              type="submit"
              disabled={isLoading || !password.trim()}
              className="flex-1 gap-2"
            >
              {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
              {t("share_consumer.unlock")}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
};
