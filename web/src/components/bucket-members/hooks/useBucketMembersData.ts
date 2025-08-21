import { useEffect, useState } from "react";
import useSWR from "swr";
import type {
  IBucket,
  IBucketMember,
} from "@/components/bucket-view/helpers/types";
import { EMAIL_REGEX } from "@/components/add-members/helpers/constants";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { api_updateMembers } from "@/components/bucket-members/helpers/api";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";

import { fetchApi } from "@/lib/api";

export interface IMemberState {
  email: string;
  group: string;
  first_name?: string;
  last_name?: string;
  status: "active" | "invited";
  isNew?: boolean;
}

export const useBucketMembersData = (bucket: IBucket) => {
  const { data, isLoading, mutate } = useSWR(
    bucket.id ? `/buckets/${bucket.id}/members` : null,
    fetchApi<{ data: Array<IBucketMember> }>,
  );

  const { session } = useSessionContext();

  const [membersState, setMembersState] = useState<Array<IMemberState>>([]);
  const [newMemberEmail, setNewMemberEmail] = useState("");
  const [newMemberGroup, setNewMemberGroup] = useState("viewer");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const currentUserEmail = session?.loggedUser?.email;
  const currentUserName = `${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`;

  useEffect(() => {
    if (data?.data) {
      setMembersState(
        data.data.map((member) => ({
          email: member.email,
          group: member.group,
          first_name: member.first_name,
          last_name: member.last_name,
          status: member.status,
          isNew: false,
        })),
      );
    }
  }, [data]);

  const originalMembersMap = new Map(data?.data.map((m) => [m.email, m.group]));

  const hasChanges =
    membersState.some(
      (member) =>
        member.isNew || originalMembersMap.get(member.email) !== member.group,
    ) || originalMembersMap.size !== membersState.length;

  const addMember = () => {
    if (!newMemberEmail.trim() || !EMAIL_REGEX.test(newMemberEmail)) return;

    const existingMember = membersState.find((m) => m.email === newMemberEmail);
    if (existingMember) {
      toast({
        variant: "destructive",
        description: "User is already a member of this bucket",
      });
      return;
    }

    setMembersState((prev) => [
      ...prev,
      {
        email: newMemberEmail.trim(),
        group: newMemberGroup,
        status: "invited" as const,
        isNew: true,
      },
    ]);

    setNewMemberEmail("");
    setNewMemberGroup("viewer");
  };

  const updateMemberRole = (email: string, newGroup: string) => {
    if (email === currentUserEmail) return;

    if (newGroup === "remove") {
      setMembersState((prev) => prev.filter((m) => m.email !== email));
    } else {
      setMembersState((prev) =>
        prev.map((m) => (m.email === email ? { ...m, group: newGroup } : m)),
      );
    }
  };

  const handleUpdateMembers = () => {
    if (!hasChanges) return;

    setIsSubmitting(true);

    const membersList = membersState.map((member) => ({
      email: member.email,
      group: member.group,
    }));

    api_updateMembers(bucket.id, membersList)
      .then(() => {
        setNewMemberEmail("");
        setNewMemberGroup("viewer");
        mutate();
        successToast("Bucket members updated successfully");
      })
      .catch(errorToast)
      .finally(() => setIsSubmitting(false));
  };

  return {
    isLoading,
    membersState,
    newMemberEmail,
    setNewMemberEmail,
    newMemberGroup,
    setNewMemberGroup,
    currentUserEmail,
    currentUserName,
    hasChanges,
    isSubmitting,
    addMember,
    updateMemberRole,
    handleUpdateMembers,
  };
};
