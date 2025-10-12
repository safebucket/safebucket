import { useState } from "react";
import type { IMembers } from "@/components/bucket-view/helpers/types.ts";
import { EMAIL_REGEX } from "@/types/bucket.ts";

export const useAddMembers = (
  initialShareWith: Array<IMembers> = [],
  onShareWithChange?: (shareWith: Array<IMembers>) => void,
) => {
  const [email, setEmail] = useState<string>("");
  const [shareWith, setShareWithState] =
    useState<Array<IMembers>>(initialShareWith);

  const updateShareWith = (newShareWith: Array<IMembers>) => {
    setShareWithState(newShareWith);
    onShareWithChange?.(newShareWith);
  };

  const addEmail = (email: string, group: string = "viewer") => {
    if (EMAIL_REGEX.test(email) && !shareWith.find((e) => e.email === email)) {
      const newShareWith = [...shareWith, { email: email, group: group }];
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
