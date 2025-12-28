import {
  Activity,
  Home,
  LayoutDashboard,
  LifeBuoy,
  Send,
  Settings2,
  Shield,
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
      ],
    },
  ],
  settings: [
    {
      title: "navigation.settings",
      url: "#",
      icon: Settings2,
      items: [
        {
          title: "navigation.profile",
          url: "/settings/profile",
        },
        {
          title: "navigation.preferences",
          url: "/settings/preferences",
        },
      ],
    },
  ],
  help: [
    {
      title: "Support",
      url: "#",
      icon: LifeBuoy,
    },
    {
      title: "Feedback",
      url: "#",
      icon: Send,
    },
  ],
};
