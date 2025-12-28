import type { FC } from "react";
import type { LucideIcon } from "lucide-react";
import { ArrowRight } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface StatCardProps {
  title: string;
  value: string | number;
  icon: LucideIcon;
  href?: string;
}

export const StatCard: FC<StatCardProps> = ({
  title,
  value,
  icon: Icon,
  href,
}) => {
  const content = (
    <Card className={href ? "transition-colors hover:bg-muted/50" : undefined}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent className="flex items-end justify-between">
        <div className="text-2xl font-bold">{value}</div>
        {href && <ArrowRight className="h-4 w-4 text-muted-foreground" />}
      </CardContent>
    </Card>
  );

  if (href) {
    return <Link to={href}>{content}</Link>;
  }

  return content;
};
