import { NotSet } from "./NotSet";
import { formatFileSize } from "@/lib/utils";

export function BytesValue({ value }: { value?: number | null }) {
  if (value === undefined || value === null) {
    return <NotSet />;
  }
  return <span>{formatFileSize(value)}</span>;
}
