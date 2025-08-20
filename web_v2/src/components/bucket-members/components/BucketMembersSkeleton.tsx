import { Users } from "lucide-react";
import type { FC } from "react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export const BucketMembersSkeleton: FC = () => {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Users className="h-5 w-5" />
          Bucket Members
        </CardTitle>
        <CardDescription>
          Manage who has access to this bucket and their permissions
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          <div className="h-4 w-24 animate-pulse rounded bg-muted" />
          <div className="flex gap-3">
            <div className="h-10 flex-1 animate-pulse rounded bg-muted" />
            <div className="h-10 w-32 animate-pulse rounded bg-muted" />
            <div className="h-10 w-10 animate-pulse rounded bg-muted" />
          </div>
        </div>

        <div className="space-y-4">
          <div className="h-4 w-16 animate-pulse rounded bg-muted" />
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="flex items-center space-x-4">
                  <div className="h-10 w-10 animate-pulse rounded-full bg-muted" />
                  <div className="space-y-2">
                    <div className="h-4 w-32 animate-pulse rounded bg-muted" />
                    <div className="h-3 w-48 animate-pulse rounded bg-muted" />
                  </div>
                </div>
                <div className="h-10 w-32 animate-pulse rounded bg-muted" />
              </div>
            ))}
          </div>
        </div>

        <div className="flex justify-end border-t pt-4">
          <div className="h-10 w-32 animate-pulse rounded bg-muted" />
        </div>
      </CardContent>
    </Card>
  );
};
