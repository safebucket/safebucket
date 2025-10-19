import { useState } from "react";
import { Check, Edit2, X } from "lucide-react";
import { Input } from "@/components/ui/input.tsx";
import { Button } from "@/components/ui/button.tsx";

interface EditableFieldProps {
  label: string;
  value: string;
  onSave: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  type?: "text" | "email";
  isLoading?: boolean;
}

export function EditableField({
  label,
  value,
  onSave,
  disabled = false,
  placeholder,
  type = "text",
  isLoading = false,
}: EditableFieldProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(value);

  const handleSave = () => {
    if (editValue.trim() !== value) {
      onSave(editValue.trim());
    }
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditValue(value);
    setIsEditing(false);
  };

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">{label}</label>
      <div className="flex items-center gap-2">
        {isEditing ? (
          <>
            <Input
              type={type}
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
              placeholder={placeholder}
              className="text-sm"
              autoFocus
              disabled={isLoading}
            />
            <Button
              size="sm"
              onClick={handleSave}
              disabled={isLoading || !editValue.trim()}
            >
              <Check className="h-3 w-3" />
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={handleCancel}
              disabled={isLoading}
            >
              <X className="h-3 w-3" />
            </Button>
          </>
        ) : (
          <>
            <Input type={type} value={value} disabled className="text-sm" />
            {!disabled && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setIsEditing(true)}
              >
                <Edit2 className="h-3 w-3" />
              </Button>
            )}
          </>
        )}
      </div>
    </div>
  );
}
