import { Home, LifeBuoy, Send, Settings2 } from "lucide-react";

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
