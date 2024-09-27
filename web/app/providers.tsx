"use client";

import React from "react";

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
      <Main>{children}</Main>
    </ThemeProvider>
  );
}
