import { useContext } from "react";
import { TimeDisplayContext } from "@/components/time-display/context/TimeDisplayProvider";

export function useTimeDisplay() {
  const context = useContext(TimeDisplayContext);

  if (!context) {
    throw new Error("useTimeDisplay must be used within a TimeDisplayProvider");
  }

  return context;
}
