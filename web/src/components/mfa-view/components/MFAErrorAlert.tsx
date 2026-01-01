import { AlertCircle } from "lucide-react";

interface MFAErrorAlertProps {
  error: string | null;
}

export function MFAErrorAlert({ error }: MFAErrorAlertProps) {
  if (!error) return null;

  return (
    <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
      <AlertCircle className="h-4 w-4" />
      {error}
    </div>
  );
}
