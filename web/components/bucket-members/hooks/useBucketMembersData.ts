import { useEffect, useState } from "react";

import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { EMAIL_REGEX } from "@/components/add-members/helpers/constants";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { api_updateMembers } from "@/components/bucket-members/helpers/api";
import { IBucket, IBucketMember } from "@/components/bucket-view/helpers/types";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";

export interface IMemberState {
  email: string;
  role: string;
  first_name?: string;
  last_name?: string;
  status: "active" | "invited";
  isNew?: boolean;
}

export const useBucketMembersData = (bucket: IBucket) => {
  const { data, isLoading, mutate } = useSWR(
    bucket.id ? `/buckets/${bucket?.id}/members` : null,
    fetchApi<{ data: IBucketMember[] }>,
  );

  const { session } = useSessionContext();

  const [membersState, setMembersState] = useState<IMemberState[]>([]);
  const [newMemberEmail, setNewMemberEmail] = useState("");
  const [newMemberRole, setNewMemberRole] = useState("viewer");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const currentUserEmail = session?.loggedUser?.email;
  const currentUserName = `${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`;

  useEffect(() => {
    if (data?.data) {
      setMembersState(
        data.data.map((member) => ({
          email: member.email,
          role: member.role,
          first_name: member.first_name,
          last_name: member.last_name,
          status: member.status,
          isNew: false,
        })),
      );
    }
  }, [data]);

  const originalMembersMap = new Map(data?.data.map((m) => [m.email, m.role]));

  const hasChanges =
    membersState.some(
      (member) =>
        member.isNew || originalMembersMap.get(member.email) !== member.role,
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
        role: newMemberRole,
        status: "invited" as const,
        isNew: true,
      },
    ]);

    setNewMemberEmail("");
    setNewMemberRole("viewer");
  };

  const updateMemberRole = (email: string, newRole: string) => {
    if (email === currentUserEmail) return;

    if (newRole === "remove") {
      setMembersState((prev) => prev.filter((m) => m.email !== email));
    } else {
      setMembersState((prev) =>
        prev.map((m) => (m.email === email ? { ...m, role: newRole } : m)),
      );
    }
  };

  const handleUpdateMembers = async () => {
    if (!hasChanges) return;

    setIsSubmitting(true);

    const membersList = membersState.map((member) => ({
      email: member.email,
      group: member.role,
    }));

    api_updateMembers(bucket.id, membersList)
      .then(() => {
        setNewMemberEmail("");
        setNewMemberRole("viewer");
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
    newMemberRole,
    setNewMemberRole,
    currentUserEmail,
    currentUserName,
    hasChanges,
    isSubmitting,
    addMember,
    updateMemberRole,
    handleUpdateMembers,
  };
};
