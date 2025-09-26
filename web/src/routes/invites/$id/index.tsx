import { createFileRoute } from "@tanstack/react-router";
import { EmailConfirmationForm } from "@/components/invites/components/EmailConfirmationForm.tsx";

export const Route = createFileRoute("/invites/$id/")({
  component: InvitePage,
});

function InvitePage() {
  const { id } = Route.useParams();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <EmailConfirmationForm invitationId={id} />
    </div>
  );
}
