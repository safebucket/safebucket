import type { FC, ReactNode } from "react";
import { SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";

interface IAppSidebarInset {
  children: ReactNode;
}

export const AppSidebarInset: FC<IAppSidebarInset> = ({
  children,
}: IAppSidebarInset) => {
  return (
    <SidebarInset className="min-w-0">
      <header className="flex h-16 shrink-0 items-center">
        <div className="px-4">
          <SidebarTrigger className="-ml-1" />
        </div>
      </header>
      {children}
    </SidebarInset>
  );
};
