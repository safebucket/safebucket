import React, { FC } from "react";
import { Users } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

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
          <div className="h-4 w-24 bg-muted rounded animate-pulse" />
          <div className="flex gap-3">
            <div className="flex-1 h-10 bg-muted rounded animate-pulse" />
            <div className="w-32 h-10 bg-muted rounded animate-pulse" />
            <div className="w-10 h-10 bg-muted rounded animate-pulse" />
          </div>
        </div>

        <div className="space-y-4">
          <div className="h-4 w-16 bg-muted rounded animate-pulse" />
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex items-center justify-between p-3 border rounded-lg">
                <div className="flex items-center space-x-4">
                  <div className="w-10 h-10 bg-muted rounded-full animate-pulse" />
                  <div className="space-y-2">
                    <div className="h-4 w-32 bg-muted rounded animate-pulse" />
                    <div className="h-3 w-48 bg-muted rounded animate-pulse" />
                  </div>
                </div>
                <div className="w-32 h-10 bg-muted rounded animate-pulse" />
              </div>
            ))}
          </div>
        </div>

        <div className="flex justify-end border-t pt-4">
          <div className="w-32 h-10 bg-muted rounded animate-pulse" />
        </div>
      </CardContent>
    </Card>
  );
};