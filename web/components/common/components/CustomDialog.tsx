"use client";

import React, { FC, ReactElement, ReactNode } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

interface ICustomDialogProps {
  title: string;
  description: string;
  submitName: string;
  trigger: ReactElement;
  children: ReactNode;
  onSubmit?: () => void;
  isOpen?: boolean;
  setIsOpen?: (open: boolean) => void;
}

export const CustomDialog: FC<ICustomDialogProps> = ({
  title,
  description,
  submitName,
  trigger,
  children,
  onSubmit,
  isOpen,
  setIsOpen,
}: ICustomDialogProps) => {
  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">{children}</div>
        <DialogFooter>
          <Button type="submit" onClick={onSubmit}>
            {submitName}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
