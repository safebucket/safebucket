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
        {
          title: "Activity",
          url: "/activity",
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
