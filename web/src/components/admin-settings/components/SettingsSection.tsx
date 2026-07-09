import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface SettingsSectionProps {
  id?: string;
  title: string;
  description?: string;
  icon?: LucideIcon;
  children: ReactNode;
}

export function SettingsSection({
  id,
  title,
  description,
  icon: Icon,
  children,
}: SettingsSectionProps) {
  return (
    <Card id={id} className="scroll-mt-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
          {title}
        </CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className="divide-y pt-0">{children}</CardContent>
    </Card>
  );
}
