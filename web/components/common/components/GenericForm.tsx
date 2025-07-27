import React, { FC } from "react";

import { FieldValues, useForm } from "react-hook-form";

import { FormField } from "@/components/dialogs/components/FormField";
import { IFormField } from "@/components/dialogs/helpers/types";
import { Button } from "@/components/ui/button";

interface IGenericFormProps {
  fields: IFormField[];
  onSubmit: (data: FieldValues) => void;
  isSubmitting?: boolean;
  submitLabel: string;
  className?: string;
  children?: React.ReactNode;
}

export const GenericForm: FC<IGenericFormProps> = ({
  fields,
  onSubmit,
  isSubmitting = false,
  submitLabel,
  className = "",
  children,
}: IGenericFormProps) => {
  const {
    register,
    control,
    formState: { errors, isValid },
    handleSubmit,
    watch,
  } = useForm({ mode: "onChange" });

  const values = watch();

  const onSubmitWrapper = (data: FieldValues) => {
    onSubmit(data);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmitWrapper)}
      className={`space-y-4 ${className}`}
    >
      {fields.map(
        (field) =>
          (!field.condition ||
            (field.condition && field.condition(values))) && (
            <FormField
              key={field.id}
              field={field}
              register={register}
              control={control}
              errors={errors}
            />
          ),
      )}

      {children}

      <Button
        type="submit"
        className="w-full"
        disabled={isSubmitting || !isValid}
      >
        {isSubmitting ? "Processing..." : submitLabel}
      </Button>
    </form>
  );
};
