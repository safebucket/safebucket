import type { FieldValues } from "react-hook-form";

export const shareFileFields = [
  { id: "files", label: "File", type: "file" as const, required: true },
  {
    id: "expiresAt",
    label: "Options",
    type: "collapsible" as const,
    defaultValue: false,
  },
  {
    id: "expiresAtDate",
    label: "Date",
    type: "datepicker" as const,
    condition: (values: FieldValues) => Boolean(values.expiresAt),
  },
];
