import { useTranslation } from "react-i18next";

import { Check, FolderOpen, Link, Settings } from "lucide-react";
import type { ElementType, FC } from "react";

import { cn } from "@/lib/utils";

type Step = 1 | 2 | 3;

interface IStepConfig {
  step: Step;
  icon: ElementType;
  label: string;
}

interface IStepIndicatorProps {
  current: Step;
}

export const StepIndicator: FC<IStepIndicatorProps> = ({ current }) => {
  const { t } = useTranslation();

  const steps: Array<IStepConfig> = [
    {
      step: 1,
      icon: FolderOpen,
      label: t("quick_share.step_scope"),
    },
    {
      step: 2,
      icon: Settings,
      label: t("quick_share.step_options"),
    },
    {
      step: 3,
      icon: Link,
      label: t("quick_share.step_link"),
    },
  ];

  return (
    <div className="flex items-center justify-center px-2">
      {steps.map(({ step, icon: Icon, label }, index) => (
        <div key={step} className="flex items-center">
          <div className="flex flex-col items-center gap-1.5">
            <div
              className={cn(
                "flex h-9 w-9 items-center justify-center rounded-full border-2 transition-colors",
                current === step &&
                  "border-primary bg-primary text-primary-foreground",
                current > step && "border-primary bg-primary/10 text-primary",
                current < step &&
                  "border-muted-foreground/30 text-muted-foreground",
              )}
            >
              {current > step ? (
                <Check className="h-4 w-4" />
              ) : (
                <Icon className="h-4 w-4" />
              )}
            </div>
            <span
              className={cn(
                "text-xs font-medium",
                current >= step ? "text-foreground" : "text-muted-foreground",
              )}
            >
              {label}
            </span>
          </div>
          {index < steps.length - 1 && (
            <div
              className={cn(
                "mb-5 mx-4 h-px w-16",
                current > step ? "bg-primary" : "bg-border",
              )}
            />
          )}
        </div>
      ))}
    </div>
  );
};
