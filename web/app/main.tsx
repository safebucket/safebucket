"use client";

import React from "react";

import { SideMenu } from "@/app/side-menu/side-menu";

import { Toaster } from "@/components/ui/toaster";

export function Main({ children }: { children: React.ReactNode }) {
  // if (status === "loading") {
  //   return <Loading />;
  // }

  return (
    <>
      {/*{session && <SideMenu />}*/}
      <SideMenu />
      {children}
      <Toaster />
    </>
  );
}
