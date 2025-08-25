import { Link } from "@tanstack/react-router";
import type { FC } from "react";

import {
  Breadcrumb,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";

interface IAppSidebarInset {
  children: React.ReactNode;
}

export const AppSidebarInset: FC<IAppSidebarInset> = ({
  children,
}: IAppSidebarInset) => {
  // const pathname = usePathname()

  // const path = pathname.split('/').filter((segment) => segment)
  const path = "/";
  const rootPath = `/${path[0]}/${path[1]}`;
  // const pathShort = path.slice(2, path.length)

  return (
    <SidebarInset>
      <header className="flex h-16 shrink-0 items-center gap-2">
        <div className="flex items-center gap-2 px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <Breadcrumb>
            <BreadcrumbList>
              <BreadcrumbLink asChild>
                <Link to={rootPath}>/</Link>
              </BreadcrumbLink>
              <BreadcrumbSeparator className="hidden md:block" />
              {/* {pathShort.map((segment, index) => {*/}
              {/*  const isLast = index === pathShort.length - 1*/}
              {/*  const link = path.slice(0, index - 1).join('/')*/}
              {/*  return isLast ? (*/}
              {/*    <BreadcrumbPage key={segment}>{segment}</BreadcrumbPage>*/}
              {/*  ) : (*/}
              {/*    <React.Fragment key={segment}>*/}
              {/*      <BreadcrumbLink asChild>*/}
              {/*        <Link to={`/${link}`}>{segment}</Link>*/}
              {/*      </BreadcrumbLink>*/}
              {/*      <BreadcrumbSeparator className="hidden md:block" />*/}
              {/*    </React.Fragment>*/}
              {/*  )*/}
              {/* })}*/}
            </BreadcrumbList>
          </Breadcrumb>
        </div>
      </header>
      {children}
    </SidebarInset>
  );
};
