import React from "react";

import { FieldValues, useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface FormField {
  id: string;
  label: string;
  type: string;
  required?: boolean;
}

interface IFormDialogProps {
  title: string;
  fields: FormField[];
  onSubmit: (data: FieldValues) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  confirmLabel: string;
}

export default function FormDialog({
  title,
  fields,
  onSubmit,
  open,
  onOpenChange,
  confirmLabel,
}: IFormDialogProps) {
  const { register, handleSubmit, reset } = useForm();

  const onSubmitWrapper = (data: FieldValues) => {
    onSubmit(data);
    reset();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmitWrapper)}>
          <div className="grid gap-4 py-4">
            {fields.map((field) => (
              <div
                key={field.id}
                className="grid grid-cols-4 items-center gap-4"
              >
                <Label htmlFor={field.id}>{field.label}</Label>
                <Input
                  id={field.id}
                  type={field.type}
                  {...register(field.id, { required: field.required })}
                  className="col-span-3"
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button type="submit">{confirmLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
