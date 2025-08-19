import { useState } from "react";

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

interface Field {
  id: string;
  label: string;
  type: string;
  required?: boolean;
}

interface FormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  fields: Field[];
  onSubmit: (data: Record<string, string>) => void;
  confirmLabel?: string;
  maxWidth?: string;
  children?: React.ReactNode;
}

export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  fields,
  onSubmit,
  confirmLabel = "Submit",
  children,
}: FormDialogProps) {
  const [formData, setFormData] = useState<Record<string, string>>({});

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
    onOpenChange(false);
    setFormData({});
  };

  const handleInputChange = (fieldId: string, value: string) => {
    setFormData((prev) => ({ ...prev, [fieldId]: value }));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="grid gap-4 py-4">
            {fields.map((field) => (
              <div
                key={field.id}
                className="grid grid-cols-4 items-center gap-4"
              >
                <Label htmlFor={field.id} className="text-right">
                  {field.label}
                </Label>
                <Input
                  id={field.id}
                  type={field.type}
                  required={field.required}
                  className="col-span-3"
                  value={formData[field.id] || ""}
                  onChange={(e) => handleInputChange(field.id, e.target.value)}
                />
              </div>
            ))}
            {children}
          </div>
          <DialogFooter>
            <Button type="submit">{confirmLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
