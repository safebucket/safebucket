"use client";

import React, { FC, useState } from "react";

import {
  ChevronsUpDown,
  FolderSync,
  LogOut,
  Plus,
  ShieldCheck,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { AddMembers } from "@/components/add-members";
import { nav } from "@/components/app-sidebar/helpers/nav";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { IInvites } from "@/components/bucket-view/helpers/types";
import { useBucketsData } from "@/components/bucket-view/hooks/useBucketsData";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";

export const AppSidebar: FC = () => {
  const pathname = usePathname();
  const { session, logout } = useSessionContext();
  const createBucketDialog = useDialog();
  const { buckets, createBucketAndInvites } = useBucketsData();

  const [shareWith, setShareWith] = useState<IInvites[]>([]);

  return (
    <Sidebar variant="inset">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <div className="mt-2 flex items-center justify-center gap-2 text-xl font-semibold text-primary">
              <ShieldCheck className="h-6 w-6" />
              <span>Safebucket</span>
            </div>
            <Separator className="mt-4" />
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarMenu>
          <SidebarGroup>
            {nav.main.map((item) => (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton asChild tooltip={item.title}>
                  <div>
                    <item.icon />
                    {item.title}
                  </div>
                </SidebarMenuButton>
                <SidebarMenuSub>
                  {item.items?.map((subItem) => (
                    <SidebarMenuSubItem key={subItem.title}>
                      <SidebarMenuSubButton
                        asChild
                        isActive={pathname == subItem.url}
                      >
                        <Link href={subItem.url}>{subItem.title}</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                  ))}
                </SidebarMenuSub>
              </SidebarMenuItem>
            ))}
          </SidebarGroup>
          <SidebarGroup>
            <SidebarMenuItem key="shared_buckets">
              <SidebarMenuButton asChild tooltip="Shared Buckets">
                <div>
                  <FolderSync />
                  Shared Buckets
                </div>
              </SidebarMenuButton>
              <SidebarMenuAction>
                <Plus onClick={createBucketDialog.trigger} />
                <FormDialog
                  {...createBucketDialog.props}
                  title="New bucket"
                  maxWidth="650px"
                  description="Create a bucket to share files safely"
                  fields={[
                    { id: "name", label: "Name", type: "text", required: true },
                  ]}
                  onSubmit={(data) => {
                    createBucketAndInvites(data.name, shareWith);
                    setShareWith([]);
                  }}
                  confirmLabel="Create"
                >
                  <AddMembers
                    shareWith={shareWith}
                    onShareWithChange={setShareWith}
                    currentUserEmail={session?.loggedUser?.email}
                    currentUserName={`${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`}
                  />
                </FormDialog>
              </SidebarMenuAction>
              <SidebarMenuSub>
                {buckets.map((bucket) => (
                  <SidebarMenuSubItem key={bucket.id}>
                    <SidebarMenuSubButton
                      asChild
                      isActive={pathname.startsWith(`/buckets/${bucket.id}`)}
                    >
                      <Link href={`/buckets/${bucket.id}`}>{bucket.name}</Link>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                ))}
              </SidebarMenuSub>
            </SidebarMenuItem>
          </SidebarGroup>
        </SidebarMenu>
        <SidebarGroup className="mt-auto">
          <SidebarGroupLabel>Help</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {nav.help.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild size="sm">
                    <Link href={item.url}>
                      <item.icon />
                      {item.title}
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton
                  size="lg"
                  className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                >
                  <Avatar className="h-8 w-8 rounded-lg">
                    <AvatarImage src={nav.user.avatar} alt="Image" />
                    <AvatarFallback className="rounded-lg">
                      {session?.loggedUser?.email.charAt(0)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-semibold">
                      {`${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`}
                    </span>
                    <span className="truncate text-xs">
                      {session?.loggedUser?.email}
                    </span>
                  </div>
                  <ChevronsUpDown className="ml-auto size-4" />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
                side="bottom"
                align="end"
                sideOffset={4}
              >
                <DropdownMenuLabel className="p-0 font-normal">
                  <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <Avatar className="h-8 w-8 rounded-lg">
                      <AvatarImage src={nav.user.avatar} alt="Image" />
                      <AvatarFallback className="rounded-lg">
                        {session?.loggedUser?.email.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-semibold">
                        {`${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`}
                      </span>
                      <span className="truncate text-xs">
                        {session?.loggedUser?.email}
                      </span>
                    </div>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={logout}>
                  <LogOut />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
};
