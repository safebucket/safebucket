"use client";

import React from "react";

import { AppSidebar } from "@/components/app-sidebar/AppSidebar";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { LoadingView } from "@/components/common/components/LoadingView";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/toaster";

export function Main({ children }: { children: React.ReactNode }) {
  const { session, status } = useSessionContext();

  if (status === "loading") {
    return <LoadingView />;
  }

  return (
    <>
      {session && <AppSidebar />}
      {/* TODO: Show bucket path */}
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2">
          <div className="flex items-center gap-2 px-4">
            <SidebarTrigger className="-ml-1" />
            <Separator orientation="vertical" className="mr-2 h-4" />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href="#">Buckets</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>/</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </header>
        {children}
      </SidebarInset>
      <Toaster />
    </>
  );
}
