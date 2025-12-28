import { useTranslation } from "react-i18next";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import type { FC } from "react";
import type { TimeSeriesPoint } from "@/queries/admin";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface SharedFilesChartProps {
  data: Array<TimeSeriesPoint>;
  timeRange: string;
  onTimeRangeChange: (value: string) => void;
}

const chartConfig = {
  count: {
    label: "Files",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig;

export const SharedFilesChart: FC<SharedFilesChartProps> = ({
  data,
  timeRange,
  onTimeRangeChange,
}) => {
  const { t } = useTranslation();

  const formattedData = data.map((point) => ({
    ...point,
    formattedDate: new Date(point.date).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
    }),
  }));

  return (
    <Card>
      <CardHeader className="flex items-center gap-2 space-y-0 border-b py-5 sm:flex-row">
        <div className="grid flex-1 gap-1 text-center sm:text-left">
          <CardTitle>{t("admin.dashboard.shared_files_chart.title")}</CardTitle>
          <CardDescription>
            {t("admin.dashboard.shared_files_chart.description")}
          </CardDescription>
        </div>
        <Select value={timeRange} onValueChange={onTimeRangeChange}>
          <SelectTrigger
            className="w-[160px] rounded-lg sm:ml-auto"
            aria-label="Select a time range"
          >
            <SelectValue placeholder="Select range" />
          </SelectTrigger>
          <SelectContent className="rounded-xl">
            <SelectItem value="180" className="rounded-lg">
              {t("admin.dashboard.shared_files_chart.6_months")}
            </SelectItem>
            <SelectItem value="90" className="rounded-lg">
              {t("admin.dashboard.shared_files_chart.3_months")}
            </SelectItem>
            <SelectItem value="30" className="rounded-lg">
              {t("admin.dashboard.shared_files_chart.1_month")}
            </SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent className="px-2 pt-4 sm:px-6 sm:pt-6">
        <ChartContainer
          config={chartConfig}
          className="aspect-auto h-[250px] w-full"
        >
          <AreaChart data={formattedData}>
            <defs>
              <linearGradient id="fillCount" x1="0" y1="0" x2="0" y2="1">
                <stop
                  offset="5%"
                  stopColor="var(--color-count)"
                  stopOpacity={0.8}
                />
                <stop
                  offset="95%"
                  stopColor="var(--color-count)"
                  stopOpacity={0.1}
                />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="formattedDate"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              minTickGap={32}
            />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  labelFormatter={(value) => value}
                  indicator="dot"
                />
              }
            />
            <Area
              dataKey="count"
              type="natural"
              fill="url(#fillCount)"
              stroke="var(--color-count)"
              stackId="a"
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
};
