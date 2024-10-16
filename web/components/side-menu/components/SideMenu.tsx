"use client";

import React, { FC } from "react";

import { LogOut, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { Settings } from "@/components/side-menu/components/Settings";
import { SharedBuckets } from "@/components/side-menu/components/SharedBuckets";
import { Button } from "@/components/ui/button";

export const SideMenu: FC = () => {
  const pathname = usePathname();
  const { logout } = useSessionContext();

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
        <Button
          variant="outline"
          size="sm"
          className="mt-4 w-full hover:bg-muted hover:text-primary"
          onClick={logout}
        >
          <LogOut className="mr-2 h-4 w-4" />
          Logout
        </Button>
      </div>
    </div>
  );
};
