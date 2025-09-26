import { createFileRoute } from "@tanstack/react-router";
import { InviteFormSmartEnrollment } from "@/components/invites/components/InviteFormSmartEnrollment.tsx";
import { authProvidersQueryOptions } from "@/queries/auth_providers.ts";

export const Route = createFileRoute("/invites/$id/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(authProvidersQueryOptions()),
  component: InvitePage,
});

function InvitePage() {
  const { id } = Route.useParams();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <InviteFormSmartEnrollment invitationId={id} />
    </div>
  );
}
