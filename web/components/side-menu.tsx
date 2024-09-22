"use client";

import React from "react";

import { House, LogOut, ShieldCheck } from "lucide-react";
import { signOut } from "next-auth/react";
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
        <div className="flex items-center justify-center gap-2 text-xl font-semibold text-primary">
          <ShieldCheck className="h-6 w-6" />
          <span>Safebucket</span>
        </div>
        <div>
          <nav className="space-y-1">
            <Link
              href="/"
              className={`flex items-center rounded-md px-3 py-2 hover:bg-muted ${pathname == "/" ? "bg-muted text-primary" : ""}`}
            >
              <House className="mr-2 h-5 w-5" />
              Home
            </Link>
          </nav>
        </div>
        <div>
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-lg font-medium">Shared buckets</h3>
            <Button
              variant="outline"
              size="sm"
              className="hover:bg-muted hover:text-primary"
            >
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
          <Button
            variant="outline"
            size="sm"
            className="mt-4 w-full hover:bg-muted hover:text-primary"
            onClick={() => signOut()}
          >
            <LogOut className="mr-2 h-4 w-4" />
            Logout
          </Button>
        </div>
      </div>
    </div>
  );
}
