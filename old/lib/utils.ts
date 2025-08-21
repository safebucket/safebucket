import { type ClassValue, clsx } from "clsx";
import { format, parseISO } from "date-fns";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(timestamp: string) {
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
