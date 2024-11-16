import { FieldValues } from "react-hook-form";

export interface IFormField {
  id: string;
  label: string;
  type: "text" | "password" | "file" | "select" | "switch" | "datepicker";
  required?: boolean;
  options?: { value: string; label: string }[];
  defaultValue?: string | boolean;
  condition?: (values: FieldValues) => boolean;
}
