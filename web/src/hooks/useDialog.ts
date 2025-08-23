import { useState } from "react";

export function useDialog() {
  const [isOpen, setIsOpen] = useState(false);

  const trigger = () => setIsOpen(true);
  const close = () => setIsOpen(false);

  const props = {
    open: isOpen,
    onOpenChange: setIsOpen,
  };

  return {
    trigger,
    close,
    props,
  };
}
