"use client";

import React, { use } from "react";

import { ChallengeValidationForm } from "@/components/invites/components/ChallengeValidationForm";

interface IValidateChallengePageProps {
  params: Promise<{ id: string; challengeId: string }>;
}

export default function ValidateChallengePage(
  props: IValidateChallengePageProps,
) {
  const params = use(props.params);

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <ChallengeValidationForm
        invitationId={params.id}
        challengeId={params.challengeId}
      />
    </div>
  );
}
