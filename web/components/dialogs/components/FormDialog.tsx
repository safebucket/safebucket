import React, { FC } from "react";

import { FieldValues, useForm } from "react-hook-form";

import { FormField } from "@/components/dialogs/components/FormField";
import { IFormField } from "@/components/dialogs/helpers/types";
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
  fields: IFormField[];
  onSubmit: (data: FieldValues) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  confirmLabel: string;
}

export const FormDialog: FC<IFormDialogProps> = ({
  title,
  description,
  fields,
  onSubmit,
  open,
  onOpenChange,
  confirmLabel,
}: IFormDialogProps) => {
  const { register, control, handleSubmit, watch, reset } = useForm();

  const values = watch();

  const onSubmitWrapper = (data: FieldValues) => {
    onSubmit(data);
    const fieldsToReset = fields.filter((field) => field.type !== "file");
    reset(fieldsToReset);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmitWrapper)}>
          <div className="grid gap-4 py-4">
            {fields.map(
              (field) =>
                (!field.condition ||
                  (field.condition && field.condition(values))) && (
                  <FormField
                    key={field.id}
                    field={field}
                    register={register}
                    control={control}
                  />
                ),
            )}
          </div>
          <DialogFooter>
            <Button type="submit">{confirmLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
