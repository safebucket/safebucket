import { NotSet } from "./NotSet";

export function TextValue({ value }: { value?: string | number | null }) {
  if (value === undefined || value === null || value === "") {
    return <NotSet />;
  }
  return <span>{value}</span>;
}
