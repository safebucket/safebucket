import type { FieldValues } from "react-hook-form";

export interface IFormField {
  id: string;
  label: string;
  type:
    | "text"
    | "password"
    | "file"
    | "select"
    | "switch"
    | "datepicker"
    | "otp";
  placeholder?: string;
  required?: boolean;
  options?: Array<{ value: string; label: string }>;
  defaultValue?: string | boolean;
  condition?: (values: FieldValues) => boolean;
  maxLength?: number;
}
