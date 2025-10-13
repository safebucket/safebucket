import { createFileRoute } from "@tanstack/react-router";
import { PasswordResetValidateForm } from "@/components/reset-password/PasswordResetValidateForm.tsx";

export const Route = createFileRoute("/auth/reset-password/$id/")({
  component: PasswordResetValidateComponent,
});

function PasswordResetValidateComponent() {
  const { id } = Route.useParams();

  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <PasswordResetValidateForm challengeId={id} />
      </div>
    </div>
  );
}
