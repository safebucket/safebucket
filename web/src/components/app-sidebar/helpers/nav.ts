import {
  Activity,
  FolderOpen,
  Home,
  LayoutDashboard,
  Shield,
  SlidersHorizontal,
  User,
  Users,
} from "lucide-react";

export const nav = {
  user: {
    avatar: "/avatars/safebucket.jpg",
  },
  main: [
    {
      title: "navigation.personal",
      url: "#",
      icon: Home,
      items: [
        {
          title: "navigation.home",
          url: "/",
        },
        {
          title: "navigation.activity",
          url: "/activity",
        },
      ],
    },
  ],
  admin: [
    {
      title: "navigation.administration",
      url: "#",
      icon: Shield,
      items: [
        {
          title: "navigation.dashboard",
          url: "/admin/dashboard",
          icon: LayoutDashboard,
        },
        {
          title: "navigation.admin_activity",
          url: "/admin/activity",
          icon: Activity,
        },
        {
          title: "navigation.users",
          url: "/admin/users",
          icon: Users,
        },
        {
          title: "navigation.buckets",
          url: "/admin/buckets",
          icon: FolderOpen,
        },
      ],
    },
  ],
  help: [
    {
      title: "navigation.account",
      url: "/settings/profile",
      icon: User,
    },
    {
      title: "navigation.preferences",
      url: "/settings/preferences",
      icon: SlidersHorizontal,
    },
  ],
};
