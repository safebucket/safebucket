import { Home, LifeBuoy, Send, Settings2 } from "lucide-react";

export const nav = {
  user: {
    avatar: "/avatars/safebucket.jpg",
  },
  main: [
    {
      title: "Personal",
      url: "#",
      icon: Home,
      items: [
        {
          title: "Home",
          url: "/",
        },
      ],
    },
  ],
  settings: [
    {
      title: "Settings",
      url: "#",
      icon: Settings2,
      items: [
        {
          title: "General",
          url: "#",
        },
        {
          title: "Team",
          url: "#",
        },
        {
          title: "Billing",
          url: "#",
        },
        {
          title: "Limits",
          url: "#",
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
