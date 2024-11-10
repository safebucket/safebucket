"use client";

import React from "react";

import { ThemeProvider } from "next-themes";

import { Main } from "@/app/main";

import { SessionProvider } from "@/components/auth-view/context/SessionProvider";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";
import { SidebarProvider } from "@/components/ui/sidebar";
import { UploadProvider } from "@/components/upload/context/UploadProvider";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <SessionProvider>
        <SidebarProvider>
          <UploadProvider>
            <BucketViewProvider>
              <Main>{children}</Main>
            </BucketViewProvider>
          </UploadProvider>
        </SidebarProvider>
      </SessionProvider>
    </ThemeProvider>
  );
}
