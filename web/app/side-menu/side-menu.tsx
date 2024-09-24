"use client";

import React from "react";

import { ShieldCheck } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { Settings } from "@/app/side-menu/settings";
import { SharedBuckets } from "@/app/side-menu/shared-buckets";

export function SideMenu() {
  const pathname = usePathname();

  return (
    <div className="h-screen w-64 border-r px-4 py-8 pr-6">
      <div className="space-y-3">
        <div className="flex items-center justify-center gap-2 text-xl font-semibold text-primary">
          <ShieldCheck className="h-6 w-6" />
          <span>Safebucket</span>
        </div>
        <div>
          <h3 className="text-lg font-medium">Personal</h3>
          <nav className="space-y-1">
            <Link
              href="/"
              className={`flex items-center rounded-md px-3 py-2 hover:bg-muted ${pathname == "/" ? "bg-muted text-primary" : ""}`}
            >
              Home
            </Link>
          </nav>
        </div>
        <SharedBuckets />
        <Settings />
      </div>
    </div>
  );
}
