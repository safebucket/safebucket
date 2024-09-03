import React from "react";

import type { Metadata } from "next";
import { Inter } from "next/font/google";

import { SideMenu } from "@/components/side-menu";
import { ThemeProvider } from "@/components/theme-provider";

import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Safebucket",
  description: "Share your files easily and securely",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <div className="flex h-svh max-h-svh w-full">
          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            <SideMenu />
            {children}
          </ThemeProvider>
        </div>
      </body>
    </html>
  );
}
