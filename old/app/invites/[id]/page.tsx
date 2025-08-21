"use client";

import React, { use } from "react";

import { EmailConfirmationForm } from "@/components/invites/components/EmailConfirmationForm";

interface InvitePageProps {
  params: Promise<{ id: string }>;
}

export default function InvitePage(props: InvitePageProps) {
  const params = use(props.params);

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <EmailConfirmationForm invitationId={params.id} />
    </div>
  );
}
