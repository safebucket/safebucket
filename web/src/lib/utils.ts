import { clsx } from "clsx";
import { format, parseISO } from "date-fns";
import { twMerge } from "tailwind-merge";
import type { ClassValue } from "clsx";

export function cn(...inputs: Array<ClassValue>) {
  return twMerge(clsx(inputs));
}

export function formatDate(timestamp: string | undefined | null) {
  if (!timestamp) {
    return "-";
  }
  const parsedDate = parseISO(timestamp);
  return format(parsedDate, "dd MMMM yyyy HH:mm");
}

export function formatFileSize(size: number) {
  if (size === 0) return "0 Bytes";

  const k = 1000;
  const units = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(size) / Math.log(k));
  const formattedSize = (size / Math.pow(k, i)).toFixed(2);

  return `${formattedSize} ${units[i]}`;
}

export function formatDurationParts(minutes: number): {
  value: number;
  unit: "days" | "hours" | "minutes";
} {
  if (minutes > 0 && minutes % 1440 === 0) {
    return { value: minutes / 1440, unit: "days" };
  }
  if (minutes > 0 && minutes % 60 === 0) {
    return { value: minutes / 60, unit: "hours" };
  }
  return { value: minutes, unit: "minutes" };
}

export function generateRandomString(length: number = 12): string {
  const chars =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

  let result = "";
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}
