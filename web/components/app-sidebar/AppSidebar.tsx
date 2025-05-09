"use client";

import React, { FC, useState } from "react";

import {
  BadgeCheck,
  Bell,
  ChevronsUpDown,
  CreditCard,
  FolderSync,
  LogOut,
  Plus,
  ShieldCheck,
  Sparkles,
  UserPlus,
  UserX,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { nav } from "@/components/app-sidebar/helpers/nav";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { IShareWith } from "@/components/bucket-view/helpers/types";
import { useBucketsData } from "@/components/bucket-view/hooks/useBucketsData";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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

const bucketGroups = [
  { id: "viewer", name: "Viewer", description: "Can view and download files" },
  {
    id: "contributor",
    name: "Contributor",
    description: "Can view, download and upload files",
  },
  {
    id: "owner",
    name: "Owner",
    description: "Can manage files and update the bucket",
  },
];

export const AppSidebar: FC = () => {
  const pathname = usePathname();
  const { session, logout } = useSessionContext();
  const createBucketDialog = useDialog();
  const { buckets, createBucket } = useBucketsData();

  const [email, setEmail] = useState<string>("");
  const [shareWith, setShareWith] = useState<IShareWith[]>([]);

  const addEmail = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

    if (emailRegex.test(email) && !shareWith.find((e) => e.email === email)) {
      setShareWith([...shareWith, { email: email, group: "viewer" }]);
      setEmail("");
    }
  };

  const setGroup = (email: string, groupId: string) => {
    const updated = shareWith.map((obj) =>
      obj.email === email ? { ...obj, group: groupId } : obj,
    );
    setShareWith(updated);
  };

  const removeFromList = (emailToRemove: string) => {
    setShareWith((prev) => prev.filter(({ email }) => emailToRemove !== email));
  };

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
                  description="Create a bucket to share files safely"
                  fields={[
                    { id: "name", label: "Name", type: "text", required: true },
                  ]}
                  onSubmit={(data) => {
                    createBucket(data.name, shareWith);
                    setEmail("");
                    setShareWith([]);
                  }}
                  confirmLabel="Create"
                >
                  <div className="grid grid-cols-12 items-center gap-4">
                    <Label htmlFor="share_with2" className="col-span-2">
                      Share with
                    </Label>
                    <div className="col-span-10 flex space-x-2">
                      <Input
                        type="text"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                      />
                      <Button
                        variant="secondary"
                        className="shrink-0"
                        onClick={(e) => {
                          e.preventDefault();
                          addEmail(email);
                        }}
                      >
                        <UserPlus className="h-4 w-4" />
                        Add
                      </Button>
                    </div>
                  </div>

                  <Separator className="my-4" />

                  <div className="mb-2 space-y-4">
                    <div className="text-sm font-medium">
                      People with access
                    </div>
                    <div className="grid gap-2">
                      <div className="flex items-center justify-between space-x-4">
                        <div className="flex items-center space-x-4">
                          <Avatar>
                            <AvatarImage src="/avatars/03.png" />
                            <AvatarFallback>OM</AvatarFallback>
                          </Avatar>
                          <div>
                            <p className="text-sm font-medium leading-none">
                              Milou (you)
                            </p>
                            <p className="text-sm text-muted-foreground">
                              milou@safebucket.com
                            </p>
                          </div>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          className="ml-auto"
                          disabled={true}
                        >
                          Owner
                        </Button>
                      </div>
                    </div>
                  </div>

                  {shareWith.map((user) => (
                    <div
                      key={user.email}
                      className="mb-2 grid grid-cols-12 items-center"
                    >
                      <div className="col-span-8 flex items-center space-x-4">
                        <Avatar>
                          <AvatarImage src="/avatars/01.png" alt="Image" />
                          <AvatarFallback>
                            {user.email.charAt(0).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>

                        <div className="text-sm font-medium leading-none">
                          {user.email}
                        </div>
                      </div>

                      <div className="col-span-3 mr-1 flex">
                        <Select
                          value={user.group}
                          onValueChange={(val) => setGroup(user.email, val)}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {bucketGroups.map((group) => (
                              <SelectItem key={group.id} value={group.id}>
                                {group.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>

                      <div className="col-span-1">
                        <Button
                          variant="secondary"
                          onClick={(e) => {
                            e.preventDefault();
                            removeFromList(user.email);
                          }}
                        >
                          <UserX className="h-2 w-2" />
                        </Button>
                      </div>
                    </div>
                  ))}
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
          <SidebarGroup>
            {nav.settings.map((item) => (
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
                      <SidebarMenuSubButton asChild>
                        <Link href={subItem.url}>{subItem.title}</Link>
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
                    <AvatarImage src={nav.user.avatar} alt={nav.user.name} />
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
                      <AvatarImage src={nav.user.avatar} alt={nav.user.name} />
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
                <DropdownMenuGroup>
                  <DropdownMenuItem>
                    <Sparkles />
                    Upgrade to Pro
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuItem>
                    <BadgeCheck />
                    Account
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <CreditCard />
                    Billing
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Bell />
                    Notifications
                  </DropdownMenuItem>
                </DropdownMenuGroup>
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
