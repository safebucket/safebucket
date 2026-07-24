import { useTranslation } from "react-i18next";
import type { CoverageStatus } from "@/types/admin";
import { Badge } from "@/components/ui/badge";

const variants: Record<
  CoverageStatus,
  "default" | "destructive" | "outline" | "secondary"
> = {
  covered: "default",
  not_covered: "destructive",
  not_applicable: "outline",
  unknown: "secondary",
};

const labelKeys: Record<CoverageStatus, string> = {
  covered: "admin.settings.coverage.covered",
  not_covered: "admin.settings.coverage.not_covered",
  not_applicable: "admin.settings.coverage.not_applicable",
  unknown: "admin.settings.coverage.unknown",
};

export function CoverageValue({ status }: { status: CoverageStatus }) {
  const { t } = useTranslation();

  return <Badge variant={variants[status]}>{t(labelKeys[status])}</Badge>;
}
