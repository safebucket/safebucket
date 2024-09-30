"use client";

import React from "react";

import { useSession } from "@/app/auth/hooks/useSession";
import { SideMenu } from "@/app/side-menu/side-menu";

import { Loading } from "@/components/loading";
import { Toaster } from "@/components/ui/toaster";

export function Main({ children }: { children: React.ReactNode }) {
  const { session, status } = useSession();

  if (status === "loading") {
    return <Loading />;
  }

  return (
    <>
      {session && <SideMenu />}
      {children}
      <Toaster />
    </>
  );
}
