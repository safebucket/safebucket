import { Navigate, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/settings/")({
  component: Settings,
});

function Settings() {
  return <Navigate to="/settings/profile" />;
}
