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
    | "otp"
    | "date"
    | "number"
    | "section";
  placeholder?: string;
  required?: boolean;
  options?: Array<{ value: string; label: string }>;
  defaultValue?: string | boolean | number;
  condition?: (values: FieldValues) => boolean;
  maxLength?: number;
  min?: string | number;
  max?: string | number;
}
