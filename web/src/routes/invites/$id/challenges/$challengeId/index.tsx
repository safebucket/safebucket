import { createFileRoute } from "@tanstack/react-router";
import { InviteAcceptForm } from "@/components/invites/components/InviteAcceptForm.tsx";

export const Route = createFileRoute("/invites/$id/challenges/$challengeId/")({
  component: ChallengePage,
});

function ChallengePage() {
  const { id, challengeId } = Route.useParams();

  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <InviteAcceptForm invitationId={id} challengeId={challengeId} />
      </div>
    </div>
  );
}
