import { useState } from "react";

export function useDialog() {
  const [isOpen, setIsOpen] = useState(false);

  const trigger = () => setIsOpen(true);

  return {
    props: {
      open: isOpen,
      onOpenChange: setIsOpen,
    },
    trigger: trigger,
    dismiss: () => setIsOpen(false),
  };
}
