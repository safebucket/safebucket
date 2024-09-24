"use client";

import React from "react";

import { useSession } from "next-auth/react";

import { Loading } from "@/components/loading";
import { SideMenu } from "@/app/side-menu/side-menu";
import { Toaster } from "@/components/ui/toaster";

export function Main({ children }: { children: React.ReactNode }) {
  const { data: session, status } = useSession();

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
