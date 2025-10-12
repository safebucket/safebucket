import { createContext, useEffect, useState } from "react";
import type { FC, ReactNode } from "react";
import type { ColorTheme } from "@/components/theme/helpers/themes";
import { loadTheme } from "@/components/theme/helpers/themes";

type ThemeMode = "light" | "dark" | "system";

interface IThemeContext {
  mode: ThemeMode;
  colorTheme: ColorTheme;
  setMode: (mode: ThemeMode) => void;
  setColorTheme: (theme: ColorTheme) => void;
  actualTheme: "light" | "dark";
}

export const ThemeContext = createContext<IThemeContext | undefined>(undefined);

interface IThemeProviderProps {
  children: ReactNode;
}

export const ThemeProvider: FC<IThemeProviderProps> = ({ children }) => {
  const [mode, setModeState] = useState<ThemeMode>(() => {
    const saved = localStorage.getItem("theme-mode");
    return (saved as ThemeMode) || "system";
  });

  const [colorTheme, setColorThemeState] = useState<ColorTheme>(() => {
    const saved = localStorage.getItem("color-theme");
    return (saved as ColorTheme) || "default";
  });

  const [actualTheme, setActualTheme] = useState<"light" | "dark">("light");

  useEffect(() => {
    const getSystemTheme = (): "light" | "dark" => {
      return window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";
    };

    const updateTheme = () => {
      const theme = mode === "system" ? getSystemTheme() : mode;
      setActualTheme(theme);

      const root = document.documentElement;
      root.classList.remove("light", "dark");
      root.classList.add(theme);
    };

    updateTheme();

    if (mode === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const handler = () => updateTheme();
      mediaQuery.addEventListener("change", handler);
      return () => mediaQuery.removeEventListener("change", handler);
    }
  }, [mode]);

  // Load color theme CSS
  useEffect(() => {
    loadTheme(colorTheme).catch((error) => {
      console.error("Failed to load theme:", error);
    });
  }, [colorTheme]);

  const setMode = (newMode: ThemeMode) => {
    setModeState(newMode);
    localStorage.setItem("theme-mode", newMode);
  };

  const setColorTheme = (newTheme: ColorTheme) => {
    setColorThemeState(newTheme);
    localStorage.setItem("color-theme", newTheme);
  };

  return (
    <ThemeContext.Provider
      value={{ mode, colorTheme, setMode, setColorTheme, actualTheme }}
    >
      {children}
    </ThemeContext.Provider>
  );
};
