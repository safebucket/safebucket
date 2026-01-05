import { AlertCircle } from "lucide-react";

interface FormErrorAlertProps {
  error: string | null;
}

export function FormErrorAlert({ error }: FormErrorAlertProps) {
  if (!error) return null;

  return (
    <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
      <AlertCircle className="h-4 w-4 flex-shrink-0" />
      {error}
    </div>
  );
}
