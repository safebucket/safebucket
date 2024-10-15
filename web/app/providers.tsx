"use client";

import React from "react";

import { ThemeProvider } from "next-themes";

import { SessionProvider } from "@/app/auth/hooks/useSession";
import { Main } from "@/app/main";

import { UploadProvider } from "@/components/upload/UploadProvider";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <SessionProvider>
        <UploadProvider>
          <Main>{children}</Main>
        </UploadProvider>
      </SessionProvider>
    </ThemeProvider>
  );
}
