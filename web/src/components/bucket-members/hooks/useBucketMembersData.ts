import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import type { IBucket } from "@/types/bucket.ts";
import { EMAIL_REGEX } from "@/types/bucket.ts";
import { getUserDisplayName } from "@/types/user.ts";
import { bucketMembersQueryOptions } from "@/queries/bucket";
import { useCurrentUser } from "@/queries/user";
import { api } from "@/lib/api";

import { errorToast } from "@/lib/toast";

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

  const currentUserEmail = user?.email;
  const currentUserName = getUserDisplayName(user, "");

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

  const updateMembersMutation = useMutation({
    mutationFn: (variables: {
      members: Array<{ email: string; group: string }>;
      successMessage: string;
    }) =>
      api.put(`/buckets/${bucket.id}/members`, { members: variables.members }),
    onSuccess: (_data, variables) => {
      toast.success(variables.successMessage);
    },
    onError: (error) => {
      errorToast(error);
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucket.id, "members"],
      });
    },
  });

  const persist = (
    nextMembers: Array<IMemberState>,
    successMessage: string,
  ) => {
    setMembersState(nextMembers);
    updateMembersMutation.mutate({
      members: nextMembers.map((member) => ({
        email: member.email,
        group: member.group,
      })),
      successMessage,
    });
  };

  const addMember = () => {
    if (!newMemberEmail.trim() || !EMAIL_REGEX.test(newMemberEmail)) return;

    const existingMember = membersState.find((m) => m.email === newMemberEmail);
    if (existingMember) {
      errorToast(t("toast.user_already_member"));
      return;
    }

    persist(
      [
        ...membersState,
        {
          email: newMemberEmail.trim(),
          group: newMemberGroup,
          status: "invited" as const,
          isNew: true,
        },
      ],
      t("bucket.view.members.member_added"),
    );

    setNewMemberEmail("");
    setNewMemberGroup("viewer");
  };

  const updateMemberRole = (email: string, newGroup: string) => {
    if (email === currentUserEmail) return;

    persist(
      membersState.map((m) =>
        m.email === email ? { ...m, group: newGroup } : m,
      ),
      t("bucket.view.members.role_updated"),
    );
  };

  const removeMember = (email: string) => {
    if (email === currentUserEmail) return;

    persist(
      membersState.filter((m) => m.email !== email),
      t("bucket.view.members.member_removed"),
    );
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
    isSubmitting: updateMembersMutation.isPending,
    addMember,
    updateMemberRole,
    removeMember,
  };
};
