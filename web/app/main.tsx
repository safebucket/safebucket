"use client";

import React from "react";

import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { LoadingView } from "@/components/common/components/LoadingView";
import { SideMenu } from "@/components/side-menu/components/SideMenu";
import { Toaster } from "@/components/ui/toaster";

export function Main({ children }: { children: React.ReactNode }) {
  const { session, status } = useSessionContext();

  if (status === "loading") {
    return <LoadingView />;
  }

  return (
    <>
      {session && <SideMenu />}
      {children}
      <Toaster />
    </>
  );
}
