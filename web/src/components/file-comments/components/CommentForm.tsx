import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC, KeyboardEvent, SubmitEvent } from "react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";

interface ICommentFormProps {
  initialValue?: string;
  placeholder?: string;
  submitLabel: string;
  isPending: boolean;
  onSubmit: (content: string) => Promise<unknown>;
  onCancel?: () => void;
}

export const CommentForm: FC<ICommentFormProps> = ({
  initialValue = "",
  placeholder,
  submitLabel,
  isPending,
  onSubmit,
  onCancel,
}: ICommentFormProps) => {
  const { t } = useTranslation();
  const [value, setValue] = useState(initialValue);
  const content = value.trim();

  const submit = () => {
    if (!content || isPending) return;
    onSubmit(content).then(
      () => setValue(""),
      () => undefined,
    );
  };

  const handleSubmit = (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();
    submit();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      submit();
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2">
      <Textarea
        value={value}
        placeholder={placeholder}
        className="max-h-40"
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={handleKeyDown}
      />
      <div className="flex justify-end gap-2">
        {onCancel && (
          <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
        )}
        <Button type="submit" size="sm" disabled={!content || isPending}>
          {submitLabel}
        </Button>
      </div>
    </form>
  );
};
