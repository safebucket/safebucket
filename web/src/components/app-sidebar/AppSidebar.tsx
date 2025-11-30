import { useState } from "react";
import { Link, useLocation } from "@tanstack/react-router";

import { useTranslation } from "react-i18next";
import { ChevronsUpDown, FolderSync, LogOut, Plus } from "lucide-react";
import type { FC } from "react";

import type { IMembers } from "@/components/bucket-view/helpers/types";
import { nav } from "@/components/app-sidebar/helpers/nav";
import { useLogout } from "@/hooks/useAuth";
import { useBucketsData } from "@/components/bucket-view/hooks/useBucketsData";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { useCurrentUser } from "@/queries/user";
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
import { AddMembersCard } from "@/components/app-sidebar/components/AddMembersCard.tsx";

export const AppSidebar: FC = () => {
  const location = useLocation();
  const { t } = useTranslation();
  const logout = useLogout();
  const { data: user } = useCurrentUser();
  const createBucketDialog = useDialog();
  const { buckets, createBucketMutation } = useBucketsData();

  const [shareWith, setShareWith] = useState<Array<IMembers>>([]);

  return (
    <Sidebar variant="inset">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <Link to="/" className="cursor-pointer">
              <img src="/safebucket_banner.png" alt="SafeBucket Logo" />
            </Link>
            <Separator className="mt-2" />
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
              {user?.role !== "guest" && (
                <SidebarMenuAction>
                  <Plus onClick={createBucketDialog.trigger} />
                  <FormDialog
                    {...createBucketDialog.props}
                    title={t("bucket.new_bucket_dialog.title")}
                    maxWidth="700px"
                    description={t("bucket.new_bucket_dialog.description")}
                    fields={[
                      {
                        id: "name",
                        label: t("bucket.new_bucket_dialog.name_label"),
                        type: "text",
                        required: true,
                      },
                    ]}
                    onSubmit={(data) => {
                      createBucketMutation.mutate({
                        name: data.name,
                        members: shareWith,
                      });
                      setShareWith([]);
                    }}
                    confirmLabel={t("common.create")}
                  >
                    <AddMembersCard
                      shareWith={shareWith}
                      onShareWithChange={setShareWith}
                      currentUserEmail={user?.email}
                      currentUserName={`${user?.first_name} ${user?.last_name}`}
                    />
                  </FormDialog>
                </SidebarMenuAction>
              )}
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
                        to="/buckets/$bucketId/{-$folderId}"
                        params={{ bucketId: bucket.id, folderId: undefined }}
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
                        isActive={location.pathname === subItem.url}
                      >
                        <Link to={subItem.url}>{t(subItem.title)}</Link>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                  ))}
                </SidebarMenuSub>
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
                      {user?.email.charAt(0)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-semibold">
                      {`${user?.first_name} ${user?.last_name}`}
                    </span>
                    <span className="truncate text-xs">{user?.email}</span>
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
                        {user?.email.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-semibold">
                        {`${user?.first_name} ${user?.last_name}`}
                      </span>
                      <span className="truncate text-xs">{user?.email}</span>
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
