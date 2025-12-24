import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { IBucket } from "@/types/bucket.ts";
import { EMAIL_REGEX } from "@/types/bucket.ts";
import { bucketMembersQueryOptions } from "@/queries/bucket";
import { useCurrentUser } from "@/queries/user";
import { api_updateMembers } from "@/components/bucket-members/helpers/api";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";

export interface IMemberState {
  email: string;
  group: string;
  first_name?: string;
  last_name?: string;
  status: "active" | "invited";
  isNew?: boolean;
}

export const useBucketMembersData = (bucket: IBucket) => {
  const { t } = useTranslation();
  const { data, isLoading } = useQuery({
    ...bucketMembersQueryOptions(bucket.id),
    enabled: !!bucket.id,
  });

  const queryClient = useQueryClient();
  const { data: user } = useCurrentUser();

  const [membersState, setMembersState] = useState<Array<IMemberState>>([]);
  const [newMemberEmail, setNewMemberEmail] = useState("");
  const [newMemberGroup, setNewMemberGroup] = useState("viewer");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const currentUserEmail = user?.email;
  const currentUserName = `${user?.first_name} ${user?.last_name}`;

  useEffect(() => {
    if (data) {
      setMembersState(
        data.map((member) => ({
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

  const originalMembersMap = new Map(data?.map((m) => [m.email, m.group]));

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
        queryClient.invalidateQueries({
          queryKey: ["buckets", bucket.id, "members"],
        });
        successToast(t("bucket.settings.members.updated_successfully"));
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
