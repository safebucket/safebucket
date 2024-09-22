"use client";

import React from "react";

import { SessionProvider } from "next-auth/react";
import { ThemeProvider } from "next-themes";

import { Main } from "@/app/main";

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
