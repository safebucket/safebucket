import type { LucideIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface OverviewCardProps {
  title: string;
  value: string;
  hint?: string;
  icon: LucideIcon;
}

export function OverviewCard({
  title,
  value,
  hint,
  icon: Icon,
}: OverviewCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-lg font-semibold">{value}</div>
        {hint && (
          <div className="truncate text-xs text-muted-foreground">{hint}</div>
        )}
      </CardContent>
    </Card>
  );
}
