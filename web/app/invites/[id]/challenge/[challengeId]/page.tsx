"use client";

import React from "react";
import { ChallengeValidationForm } from "@/components/invites/components/ChallengeValidationForm";

interface ValidatePageProps {
  params: Promise<{
    id: string;
    challengeId: string;
  }>;
}

export default function ValidatePage({ params }: ValidatePageProps) {
  const { id, challengeId } = React.use(params);

  const handleCodeSubmit = (code: string) => {
    console.log("Code submitted:", code, "for invitation:", id, "challenge:", challengeId);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
      <ChallengeValidationForm
        invitationId={id}
        challengeId={challengeId}
        onSubmit={handleCodeSubmit}
      />
    </div>
  );
}