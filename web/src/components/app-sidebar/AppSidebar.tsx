import { useState } from "react";
import { Link, useLocation } from "@tanstack/react-router";

import { useTranslation } from "react-i18next";
import {
  ChevronsUpDown,
  FolderSync,
  LogOut,
  Plus,
  ShieldCheck,
} from "lucide-react";
import type { FC } from "react";

import type { IMembers } from "@/components/bucket-view/helpers/types";
import { AddMembers } from "@/components/add-members";
import { nav } from "@/components/app-sidebar/helpers/nav";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
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
  const location = useLocation();
  const { t } = useTranslation();
  const { session, logout } = useSessionContext();
  const createBucketDialog = useDialog();
  const { buckets, createBucketAndInvites } = useBucketsData();

  const [shareWith, setShareWith] = useState<Array<IMembers>>([]);

  return (
    <Sidebar variant="inset">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <div className="text-primary mt-2 flex items-center justify-center gap-2 text-xl font-semibold">
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
                <SidebarMenuButton asChild tooltip={t(item.title)}>
                  <div>
                    <item.icon />
                    {t(item.title)}
                  </div>
                </SidebarMenuButton>
                <SidebarMenuSub>
                  {item.items.map((subItem) => (
                    <SidebarMenuSubItem key={subItem.title}>
                      <SidebarMenuSubButton
                        asChild
                        isActive={location.pathname == subItem.url}
                      >
                        <Link to={subItem.url}>{t(subItem.title)}</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                  ))}
                </SidebarMenuSub>
              </SidebarMenuItem>
            ))}
          </SidebarGroup>
          <SidebarGroup>
            <SidebarMenuItem key="shared_buckets">
              <SidebarMenuButton
                asChild
                tooltip={t("navigation.shared_buckets")}
              >
                <div>
                  <FolderSync />
                  {t("navigation.shared_buckets")}
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
                      isActive={location.pathname.startsWith(
                        `/buckets/${bucket.id}`,
                      )}
                    >
                      <Link
                        to="/buckets/$id/{-$path}"
                        params={{ id: bucket.id }}
                      >
                        {bucket.name}
                      </Link>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                ))}
              </SidebarMenuSub>
            </SidebarMenuItem>
          </SidebarGroup>
          <SidebarGroup>
            {nav.settings.map((item) => (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton
                  asChild
                  tooltip={item.title}
                  isActive={location.pathname.startsWith("/settings")}
                >
                  <Link to={item.url}>
                    <item.icon />
                    {t(item.title)}
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarGroup>
        </SidebarMenu>
        <SidebarGroup className="mt-auto">
          <SidebarGroupLabel>Help</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {nav.help.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild size="sm">
                    <Link to={item.url}>
                      <item.icon />
                      {item.title === "Settings"
                        ? t("navigation.settings")
                        : item.title}
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
                  <LogOut className="mr-2" />
                  {t("common.logout")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
};
