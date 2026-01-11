import { ActivityViewSkeleton } from "./ActivityViewSkeleton";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function ActivityPageSkeleton() {
  return (
    <div className="w-full flex-1 overflow-auto">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-32" />
        </div>

        <Card className="py-2">
          <CardContent className="pb-0 px-2">
            <ActivityViewSkeleton />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
