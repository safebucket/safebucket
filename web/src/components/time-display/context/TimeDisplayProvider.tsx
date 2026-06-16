import { createContext, useState } from "react";
import type { FC, ReactNode } from "react";

export type TimeDisplayMode = "local" | "utc";

interface ITimeDisplayContext {
  mode: TimeDisplayMode;
  setMode: (mode: TimeDisplayMode) => void;
}

export const TimeDisplayContext = createContext<
  ITimeDisplayContext | undefined
>(undefined);

const STORAGE_KEY = "activity-tz";

interface ITimeDisplayProviderProps {
  children: ReactNode;
}

export const TimeDisplayProvider: FC<ITimeDisplayProviderProps> = ({
  children,
}) => {
  const [mode, setModeState] = useState<TimeDisplayMode>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    return saved === "utc" ? "utc" : "local";
  });

  const setMode = (newMode: TimeDisplayMode) => {
    setModeState(newMode);
    localStorage.setItem(STORAGE_KEY, newMode);
  };

  return (
    <TimeDisplayContext.Provider value={{ mode, setMode }}>
      {children}
    </TimeDisplayContext.Provider>
  );
};
