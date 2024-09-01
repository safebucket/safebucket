"use client";

import React from "react";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { Bucket } from "@/app/buckets/helpers/types";
import { useBucketsData } from "@/app/buckets/hooks/useBucketsData";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export function SideMenu() {
  const pathname = usePathname();
  const { buckets, isLoading } = useBucketsData();

  return (
    <div className="h-screen w-64 border-r px-4 py-8 pr-6">
      <div className="space-y-4">
        <div>
          <h3 className="mb-2 text-lg font-medium">Private</h3>
          <nav className="space-y-1">
            <Link
              href="/"
              className={`block rounded-md px-3 py-2 hover:bg-muted ${pathname == "/" ? "bg-muted text-primary" : ""}`}
            >
              Home
            </Link>
            <Link
              href="/"
              className="block rounded-md px-3 py-2 hover:bg-muted"
            >
              Personal bucket
            </Link>
          </nav>
        </div>
        <div>
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-lg font-medium">Shared buckets</h3>
            <Button variant="outline" size="sm">
              New
            </Button>
          </div>
          <nav className="space-y-1">
            {isLoading && <Skeleton className="h-10" />}
            {!isLoading &&
              buckets.map((bucket: Bucket) => (
                <Link
                  key={bucket.id}
                  href={`/buckets/${bucket.id}`}
                  className={`block rounded-md px-3 py-2 hover:bg-muted ${pathname == `/buckets/${bucket.id}` ? "bg-muted text-primary" : ""}`}
                >
                  {bucket.name}
                </Link>
              ))}
          </nav>
        </div>
        <div>
          <h3 className="text-lg font-medium">Settings</h3>
          <nav className="space-y-1">
            <Link
              href="#"
              className="block rounded-md px-3 py-2 hover:bg-muted"
              prefetch={false}
            >
              Account
            </Link>
            <Link
              href="#"
              className="block rounded-md px-3 py-2 hover:bg-muted"
              prefetch={false}
            >
              Notifications
            </Link>
            <Link
              href="#"
              className="block rounded-md px-3 py-2 hover:bg-muted"
              prefetch={false}
            >
              Security
            </Link>
          </nav>
        </div>
      </div>
    </div>
  );
}
