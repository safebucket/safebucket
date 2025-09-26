import { createFileRoute } from "@tanstack/react-router";
import { ChallengeValidationForm } from "@/components/invites/components/ChallengeValidationForm.tsx";

export const Route = createFileRoute("/invites/$id/challenges/$challengeId/")({
  component: ChallengePage,
});

function ChallengePage() {
  const { id, challengeId } = Route.useParams();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <ChallengeValidationForm invitationId={id} challengeId={challengeId} />
    </div>
  );
}
