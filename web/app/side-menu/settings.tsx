import React, { FC } from "react";

import { LogOut } from "lucide-react";
import Link from "next/link";

import { Button } from "@/components/ui/button";

export const Settings: FC = () => {
  return (
    <div>
      <h3 className="text-lg font-medium">Settings</h3>
      <nav className="space-y-1">
        <Link
          href="#"
          className="block rounded-md px-3 py-2 hover:bg-muted"
          prefetch={false}
        >
          Account
        </Link>
        <Link
          href="#"
          className="block rounded-md px-3 py-2 hover:bg-muted"
          prefetch={false}
        >
          Notifications
        </Link>
        <Link
          href="#"
          className="block rounded-md px-3 py-2 hover:bg-muted"
          prefetch={false}
        >
          Security
        </Link>
      </nav>
    </div>
  );
};
