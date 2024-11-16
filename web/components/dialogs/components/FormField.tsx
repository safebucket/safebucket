import React, { FC } from "react";

import {
  Control,
  Controller,
  FieldValues,
  UseFormRegister,
} from "react-hook-form";

import { Datepicker } from "@/components/common/components/Datepicker";
import { IFormField } from "@/components/dialogs/helpers/types";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

interface IFormFieldProps {
  field: IFormField;
  register: UseFormRegister<FieldValues>;
  control: Control;
}

export const FormField: FC<IFormFieldProps> = ({
  field,
  register,
  control,
}: IFormFieldProps) => {
  switch (field.type) {
    case "select":
      return (
        <div key={field.id} className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor={field.id}>{field.label}</Label>
          <Controller
            name={field.id}
            control={control}
            defaultValue={field.defaultValue}
            render={({ field: { onChange, value } }) => (
              <Select onValueChange={onChange} defaultValue={value}>
                <SelectTrigger className="col-span-3">
                  <SelectValue placeholder={field.label} />
                </SelectTrigger>
                <SelectContent>
                  {field.options?.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
        </div>
      );
    case "switch":
      return (
        <div key={field.id} className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor={field.id}>{field.label}</Label>
          <Controller
            name={field.id}
            control={control}
            defaultValue={field.defaultValue as boolean}
            render={({ field: { onChange, value } }) => (
              <Switch
                id={field.id}
                checked={value}
                onCheckedChange={onChange}
                className="col-span-3"
              />
            )}
          />
        </div>
      );
    case "datepicker":
      return (
        <div key={field.id} className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor={field.id}>
            {field.label}
          </Label>
          <Controller
            name={field.id}
            control={control}
            render={(_) => <Datepicker />}
          />
        </div>
      );
    default:
      return (
        <div key={field.id} className="grid grid-cols-4 items-center gap-4">
          <Label htmlFor={field.id}>{field.label}</Label>
          <Input
            id={field.id}
            type={field.type}
            defaultValue={field.defaultValue as string}
            className="col-span-3"
            {...register(field.id, { required: field.required })}
          />
        </div>
      );
  }
};
