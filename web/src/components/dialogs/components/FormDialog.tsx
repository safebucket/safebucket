import { useEffect } from "react";
import { useForm } from "react-hook-form";
import type { FC, ReactNode } from "react";
import type { FieldValues } from "react-hook-form";

import type { IFormField } from "@/components/dialogs/helpers/types";
import { FormField } from "@/components/dialogs/components/FormField";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface IFormDialogProps {
  title: string;
  description?: string;
  fields: Array<IFormField>;
  onSubmit: (data: FieldValues) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  confirmLabel: string;
  children?: ReactNode;
  maxWidth?: string;
}

export const FormDialog: FC<IFormDialogProps> = ({
  title,
  description,
  fields,
  onSubmit,
  open,
  onOpenChange,
  confirmLabel,
  children,
  maxWidth = "500px",
}: IFormDialogProps) => {
  const {
    register,
    control,
    formState: { errors },
    handleSubmit,
    watch,
    reset,
    clearErrors,
  } = useForm();

  useEffect(() => {
    reset();
    clearErrors();
  }, [open, onOpenChange, reset, clearErrors]);

  const values = watch();

  const onSubmitWrapper = (data: FieldValues) => {
    onSubmit(data);
    reset();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={`max-w-[${maxWidth}]`}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmitWrapper)}>
          <div className="grid gap-4 py-4">
            {fields.map(
              (field) =>
                (!field.condition || field.condition(values)) && (
                  <FormField
                    key={field.id}
                    field={field}
                    register={register}
                    control={control}
                    errors={errors}
                  />
                ),
            )}
          </div>

          {children}

          <DialogFooter>
            <Button type="submit">{confirmLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
