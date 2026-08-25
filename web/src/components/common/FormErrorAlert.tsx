import { AlertCircle } from "lucide-react";

interface FormErrorAlertProps {
  error: string | null;
}

export function FormErrorAlert({ error }: FormErrorAlertProps) {
  if (!error) return null;

  return (
    <div className="flex items-center gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
      <AlertCircle className="h-4 w-4 flex-shrink-0" />
      {error}
    </div>
  );
}
