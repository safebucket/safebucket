"use client";

import React from "react";

import { ThemeProvider } from "next-themes";

import { Main } from "@/app/main";
import { SessionProvider } from "@/app/auth/hooks/useSession";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <SessionProvider>
        <Main>{children}</Main>
      </SessionProvider>
    </ThemeProvider>
  );
}
