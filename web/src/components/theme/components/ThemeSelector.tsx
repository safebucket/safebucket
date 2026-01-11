import { Check } from "lucide-react";
import type { FC } from "react";
import type { ColorTheme } from "@/components/theme/helpers/themes";
import { themes } from "@/components/theme/helpers/themes";
import { ThemePreview } from "@/components/theme/components/ThemePreview";
import { useTheme } from "@/components/theme/hooks/useTheme";
import { cn } from "@/lib/utils";

interface IThemeSelectorProps {
  value: ColorTheme;
  onChange: (theme: ColorTheme) => void;
}

export const ThemeSelector: FC<IThemeSelectorProps> = ({ value, onChange }) => {
  const { actualTheme } = useTheme();

  return (
    <div className="grid grid-cols-3 gap-3">
      {(Object.keys(themes) as Array<ColorTheme>).map((themeKey) => {
        const theme = themes[themeKey];
        const isSelected = value === themeKey;

        return (
          <button
            key={themeKey}
            type="button"
            onClick={() => onChange(themeKey)}
            className={cn(
              "flex flex-col items-center gap-2 rounded-lg border-2 p-3 transition-all hover:bg-accent/50",
              isSelected
                ? "border-primary bg-accent/30"
                : "border-border bg-transparent",
            )}
          >
            <div className="relative">
              <ThemePreview
                theme={theme}
                mode={actualTheme}
                className="h-10 w-10"
              />
              {isSelected && (
                <div className="bg-primary absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full">
                  <Check className="h-3 w-3 text-white" />
                </div>
              )}
            </div>
            <span className="text-sm font-medium">{theme.label}</span>
          </button>
        );
      })}
    </div>
  );
};
