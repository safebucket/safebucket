import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { LabelList, Pie, PieChart } from "recharts";
import type { FC } from "react";
import type { ProviderCount } from "@/queries/admin";
import type { ChartConfig } from "@/components/ui/chart";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface AuthProviderChartProps {
  data: Array<ProviderCount>;
}

const PROVIDER_COLORS: Record<string, string> = {
  local: "var(--chart-4)",
  oidc: "var(--chart-5)",
};

export const AuthProviderChart: FC<AuthProviderChartProps> = ({ data }) => {
  const { t } = useTranslation();

  const chartConfig = useMemo(() => {
    const config: ChartConfig = {};
    for (const item of data) {
      config[item.provider] = {
        label: t(`admin.dashboard.providers.${item.provider}`),
        color: PROVIDER_COLORS[item.provider] ?? "var(--chart-3)",
      };
    }
    return config;
  }, [data, t]);

  const chartData = useMemo(() => {
    return data.map((item) => ({
      name: t(`admin.dashboard.providers.${item.provider}`),
      value: item.count,
      fill: PROVIDER_COLORS[item.provider] ?? "var(--chart-3)",
    }));
  }, [data, t]);

  return (
    <Card className="flex flex-col">
      <CardHeader className="items-center pb-0">
        <CardTitle>
          {t("admin.dashboard.provider_distribution_chart.title")}
        </CardTitle>
        <CardDescription>
          {t("admin.dashboard.provider_distribution_chart.description")}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex-1 pb-0">
        <ChartContainer
          config={chartConfig}
          className="mx-auto aspect-square max-h-[200px]"
        >
          <PieChart>
            <ChartTooltip
              cursor={false}
              content={<ChartTooltipContent hideLabel />}
            />
            <Pie data={chartData} dataKey="value" nameKey="name">
              <LabelList
                dataKey="name"
                className="fill-background"
                stroke="none"
                fontSize={12}
              />
            </Pie>
          </PieChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
};
