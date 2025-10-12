import type { FC } from "react";
import type { ThemeConfig } from "@/components/theme/helpers/themes";
import { cn } from "@/lib/utils";

interface IThemePreviewProps {
  theme: ThemeConfig;
  mode?: "light" | "dark";
  className?: string;
}

export const ThemePreview: FC<IThemePreviewProps> = ({
  theme,
  mode = "light",
  className,
}) => {
  const colors =
    mode === "dark" ? theme.previewColors.dark : theme.previewColors.light;

  return (
    <div
      className={cn(
        "grid grid-cols-2 gap-1 overflow-hidden rounded-md",
        className,
      )}
    >
      <div
        className="h-4 w-4 rounded-tl-sm"
        style={{ backgroundColor: colors.primary }}
        title="Primary"
      />
      <div
        className="h-4 w-4 rounded-tr-sm"
        style={{ backgroundColor: colors.secondary }}
        title="Secondary"
      />
      <div
        className="h-4 w-4 rounded-bl-sm"
        style={{ backgroundColor: colors.accent }}
        title="Accent"
      />
      <div
        className="h-4 w-4 rounded-br-sm"
        style={{ backgroundColor: colors.muted }}
        title="Muted"
      />
    </div>
  );
};
