import { useState } from "react";

import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import type {
  IBucket,
  IBucketMember,
  IMembers,
} from "@/components/bucket-view/helpers/types";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { api_updateMembers } from "@/components/upload/helpers/api";

export interface IBucketMembersData {
  members: IBucketMember[];
  membersError: string;
  isMembersLoading: boolean;
  shareWith: IMembers[];
  setShareWith: (invites: IMembers[]) => void;
  allMembers: IMembers[];
  setAllMembers: (members: IMembers[]) => void;
  hasChanges: boolean;
  currentUserEmail: string | undefined;
  currentUserName: string;
  handleUpdateMembers: () => Promise<void>;
}

export const useBucketMembers = (bucket: IBucket): IBucketMembersData => {
  const {
    data: membersData,
    error: membersError,
    isLoading: isMembersLoading,
    mutate,
  } = useSWR(
    bucket?.id ? `/buckets/${bucket.id}/members` : null,
    fetchApi<{ data: IBucketMember[] }>,
  );

  const [shareWith, setShareWith] = useState<IMembers[]>([]);
  const [allMembers, setAllMembers] = useState<IMembers[]>([]);

  const { session } = useSessionContext();

  const hasChanges = shareWith.length > 0 || allMembers.length > 0;
  const currentUserEmail = session?.loggedUser?.email;
  const currentUserName = `${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`;

  const handleUpdateMembers = async () => {
    api_updateMembers(bucket.id, allMembers)
      .then(() => {
        setShareWith([]);
        mutate();
        successToast("Bucket members have been updated");
      })
      .catch(errorToast);
  };

  return {
    members: membersData?.data ?? [],
    membersError,
    isMembersLoading,
    shareWith,
    setShareWith,
    allMembers,
    setAllMembers,
    hasChanges,
    currentUserEmail,
    currentUserName,
    handleUpdateMembers,
  };
};
