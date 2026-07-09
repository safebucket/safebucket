import { NotSet } from "./NotSet";
import { Badge } from "@/components/ui/badge";

export function ListValue({ values }: { values?: Array<string> }) {
  if (!values || values.length === 0) {
    return <NotSet />;
  }
  return (
    <>
      {values.map((value) => (
        <Badge key={value} variant="outline" className="font-normal">
          {value}
        </Badge>
      ))}
    </>
  );
}
