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
      url: "/settings",
      icon: Settings2,
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
