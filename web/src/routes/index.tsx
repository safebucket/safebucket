import { createFileRoute } from "@tanstack/react-router";

import { BarChart3, FileText, Play } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button.tsx";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext.ts";
import { bucketsActivityQueryOptions } from "@/queries/bucket.ts";
import { ActivityItem } from "@/components/activity-view/components/ActivityItem.tsx";

export const Route = createFileRoute("/")({
  component: Homepage,
});

function Homepage() {
  const { data: activity } = useSuspenseQuery(bucketsActivityQueryOptions());

  const { t } = useTranslation();
  const { session } = useSessionContext();

  return (
    <div className="">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="space-y-8">
          <div className="text-center">
            <h1 className="text-3xl font-bold text-foreground">
              {t("homepage.welcome", {
                firstName: session?.loggedUser?.first_name,
              })}
            </h1>
            <p className="text-lg text-muted-foreground">
              {t("homepage.subtitle")}
            </p>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <Card className="lg:col-span-2 gap-4">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileText className="w-5 h-5" />
                  {t("homepage.recent_activity.title")}
                </CardTitle>
                <CardDescription>
                  {t("homepage.recent_activity.description")}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {activity.length > 0 ? (
                    activity.slice(0, 3).map((item, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between pl-4 rounded-lg border"
                      >
                        <ActivityItem item={item} />
                      </div>
                    ))
                  ) : (
                    <div className="flex flex-1 items-center justify-center min-h-[300px]">
                      <p className="text-center text-muted-foreground">
                        {t("activity.no_activity_yet")}
                      </p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>{t("homepage.quick_start.title")}</CardTitle>
                  <CardDescription>
                    {t("homepage.quick_start.description")}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                  {[
                    t("homepage.quick_start.steps.create_bucket"),
                    t("homepage.quick_start.steps.add_members"),
                    t("homepage.quick_start.steps.share_files"),
                  ].map((step, index) => (
                    <div key={index} className="flex items-center gap-3">
                      <div className="w-6 h-6 bg-primary text-primary-foreground rounded-full flex items-center justify-center text-sm font-medium">
                        {index + 1}
                      </div>
                      <span className="text-sm text-foreground">{step}</span>
                    </div>
                  ))}
                  <div className="pt-2 border-t border-border">
                    <Button
                      variant="default"
                      className="w-full"
                      onClick={() =>
                        window.open("https://docs.safebucket.io", "_blank")
                      }
                    >
                      {t("homepage.quick_start.view_documentation")}
                    </Button>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <BarChart3 className="w-5 h-5" />
                    {t("homepage.statistics.title")}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div className="flex justify-between items-center">
                      <span className="text-muted-foreground">
                        {t("homepage.statistics.total_files")}
                      </span>
                      <span className="text-2xl font-bold text-foreground">
                        1,247
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-muted-foreground">
                        {t("homepage.statistics.sharing_spaces")}
                      </span>
                      <span className="text-2xl font-bold text-foreground">
                        23
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t("homepage.tutorial.title")}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2 p-2 bg-muted rounded-lg">
                <div className="w-12 h-12 bg-primary rounded-lg flex items-center justify-center">
                  <Play className="w-6 h-6 text-primary-foreground" />
                </div>
                <div className="flex-1">
                  <h4 className="font-medium text-foreground">
                    {t("homepage.tutorial.video_title")}
                  </h4>
                  <p className="text-sm text-muted-foreground">
                    {t("homepage.tutorial.video_description")}
                  </p>
                </div>
                <Button variant="outline">
                  {t("homepage.tutorial.coming_soon")}
                  {/* <ChevronRight className="w-4 h-4 ml-2" /> */}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
