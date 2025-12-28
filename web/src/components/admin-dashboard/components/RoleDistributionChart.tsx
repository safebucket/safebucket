import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { LabelList, Pie, PieChart } from "recharts";
import type { FC } from "react";
import type { RoleCount } from "@/queries/admin";
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

interface RoleDistributionChartProps {
  data: Array<RoleCount>;
}

const ROLE_COLORS: Record<string, string> = {
  admin: "var(--chart-1)",
  user: "var(--chart-2)",
  guest: "var(--chart-3)",
};

export const RoleDistributionChart: FC<RoleDistributionChartProps> = ({
  data,
}) => {
  const { t } = useTranslation();

  const chartConfig = useMemo(() => {
    const config: ChartConfig = {};
    for (const item of data) {
      config[item.role] = {
        label: t(`admin.dashboard.roles.${item.role}`),
        color: ROLE_COLORS[item.role] ?? "var(--chart-4)",
      };
    }
    return config;
  }, [data, t]);

  const chartData = useMemo(() => {
    return data.map((item) => ({
      name: t(`admin.dashboard.roles.${item.role}`),
      value: item.count,
      fill: ROLE_COLORS[item.role] ?? "var(--chart-4)",
    }));
  }, [data, t]);

  return (
    <Card className="flex flex-col">
      <CardHeader className="items-center pb-0">
        <CardTitle>
          {t("admin.dashboard.role_distribution_chart.title")}
        </CardTitle>
        <CardDescription>
          {t("admin.dashboard.role_distribution_chart.description")}
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
