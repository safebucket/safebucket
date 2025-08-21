import { useState } from "react";

import { EMAIL_REGEX } from "@/components/add-members/helpers/constants";
import type { IMembers } from "@/components/bucket-view/helpers/types";

export const useAddMembers = (
  initialShareWith: IMembers[] = [],
  onShareWithChange?: (shareWith: IMembers[]) => void,
) => {
  const [email, setEmail] = useState<string>("");
  const [shareWith, setShareWithState] = useState<IMembers[]>(initialShareWith);

  const updateShareWith = (newShareWith: IMembers[]) => {
    setShareWithState(newShareWith);
    onShareWithChange?.(newShareWith);
  };

  const addEmail = (email: string) => {
    if (EMAIL_REGEX.test(email) && !shareWith.find((e) => e.email === email)) {
      const newShareWith = [...shareWith, { email: email, group: "viewer" }];
      updateShareWith(newShareWith);
      setEmail("");
      return true;
    }
    return false;
  };

  const setGroup = (email: string, groupId: string) => {
    const updated = shareWith.map((obj) =>
      obj.email === email ? { ...obj, group: groupId } : obj,
    );
    updateShareWith(updated);
  };

  const removeFromList = (emailToRemove: string) => {
    const newShareWith = shareWith.filter(
      ({ email }) => emailToRemove !== email,
    );
    updateShareWith(newShareWith);
  };

  const resetShareWith = () => {
    updateShareWith([]);
    setEmail("");
  };

  return {
    email,
    setEmail,
    shareWith,
    addEmail,
    setGroup,
    removeFromList,
    resetShareWith,
  };
};
