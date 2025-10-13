import { createFileRoute } from "@tanstack/react-router";
import { PasswordResetRequestForm } from "@/components/reset-password/PasswordResetRequestForm.tsx";

export const Route = createFileRoute("/auth/reset-password/")({
  component: PasswordResetRequestComponent,
});

function PasswordResetRequestComponent() {
  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6 md:p-10">
      <div className="w-full max-w-sm">
        <PasswordResetRequestForm />
      </div>
    </div>
  );
}
