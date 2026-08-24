import { createLink } from "@tanstack/react-router";
import { forwardRef } from "react";
import type { LinkComponentProps } from "@tanstack/react-router";
import type { AnchorHTMLAttributes } from "react";

import { cn } from "@/lib/utils";

const TabAnchor = forwardRef<
  HTMLAnchorElement,
  AnchorHTMLAttributes<HTMLAnchorElement>
>(({ className, children, ...props }, ref) => (
  <a
    ref={ref}
    className={cn(
      "relative shrink-0 rounded-none border-b-2 border-transparent px-3 pb-3 pt-1 text-sm text-muted-foreground transition-colors hover:text-foreground data-[status=active]:border-primary data-[status=active]:text-foreground",
      className,
    )}
    {...props}
  >
    {children}
  </a>
));
TabAnchor.displayName = "TabAnchor";

const CreatedTabLink = createLink(TabAnchor);

export const TabLink = (props: LinkComponentProps<typeof TabAnchor>) => (
  <CreatedTabLink activeOptions={{ exact: false }} {...props} />
);
