"use client";

import React from "react";
import { EmailConfirmationForm } from "@/components/invites/EmailConfirmationForm";

interface InvitePageProps {
  params: Promise<{
    id: string;
  }>;
}

export default function InvitePage({ params }: InvitePageProps) {
  const { id } = React.use(params);

  const handleEmailSubmit = (email: string) => {
    console.log("Email submitted:", email, "for invitation:", id);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
      <EmailConfirmationForm
        invitationId={id}
        onSubmit={handleEmailSubmit}
      />
    </div>
  );
}