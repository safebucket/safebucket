import { Link, useLocation } from "@tanstack/react-router";
import React from "react";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";

import type { IFolder } from "@/types/folder";
import {
  Breadcrumb,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { bucketDataQueryOptions } from "@/queries/bucket";

interface IAppSidebarInset {
  children: React.ReactNode;
}

export const AppSidebarInset: FC<IAppSidebarInset> = ({
  children,
}: IAppSidebarInset) => {
  const location = useLocation();
  const pathname = location.pathname;

  const path = pathname.split("/").filter((segment) => segment);
  const rootPath = path.length >= 2 ? `/${path[0]}/${path[1]}` : "/";
  const pathShort = path.slice(2, path.length);

  // Check if we're on a bucket route: /buckets/{bucketId}/{folderId?}
  const isBucketRoute = path[0] === "buckets" && path.length >= 2;
  const bucketId = isBucketRoute ? path[1] : null;
  const folderId = isBucketRoute && path.length >= 3 ? path[2] : null;

  // Fetch bucket data if we're on a bucket route
  const { data: bucket } = useQuery({
    ...bucketDataQueryOptions(bucketId!),
    enabled: !!bucketId,
  });

  // Build folder breadcrumb trail
  const buildFolderPath = (currentFolderId: string): Array<IFolder> => {
    if (!bucket?.folders) return [];

    const trail: Array<IFolder> = [];
    let current = bucket.folders.find((f) => f.id === currentFolderId);

    while (current) {
      trail.unshift(current);
      current = current.folder_id
        ? bucket.folders.find((f) => f.id === current!.folder_id)
        : undefined;
    }

    return trail;
  };

  const folderPath = folderId && bucket ? buildFolderPath(folderId) : [];

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

              {/* Show folder breadcrumbs for bucket routes */}
              {isBucketRoute && folderPath.length > 0 && (
                <>
                  <BreadcrumbSeparator className="hidden md:block" />
                  {folderPath.map((folder, index) => {
                    const isLast = index === folderPath.length - 1;
                    const link = `/buckets/${bucketId}/${folder.id}`;
                    return isLast ? (
                      <BreadcrumbPage key={folder.id}>
                        {folder.name}
                      </BreadcrumbPage>
                    ) : (
                      <React.Fragment key={folder.id}>
                        <BreadcrumbLink asChild>
                          <Link to={link}>{folder.name}</Link>
                        </BreadcrumbLink>
                        <BreadcrumbSeparator className="hidden md:block" />
                      </React.Fragment>
                    );
                  })}
                </>
              )}

              {/* Generic breadcrumbs for non-bucket routes */}
              {!isBucketRoute && pathShort.length > 0 && (
                <>
                  <BreadcrumbSeparator className="hidden md:block" />
                  {pathShort.map((segment, index) => {
                    const isLast = index === pathShort.length - 1;
                    const link = "/" + path.slice(0, index + 3).join("/");
                    return isLast ? (
                      <BreadcrumbPage key={segment}>{segment}</BreadcrumbPage>
                    ) : (
                      <React.Fragment key={segment}>
                        <BreadcrumbLink asChild>
                          <Link to={link}>{segment}</Link>
                        </BreadcrumbLink>
                        <BreadcrumbSeparator className="hidden md:block" />
                      </React.Fragment>
                    );
                  })}
                </>
              )}
            </BreadcrumbList>
          </Breadcrumb>
        </div>
      </header>
      {children}
    </SidebarInset>
  );
};
