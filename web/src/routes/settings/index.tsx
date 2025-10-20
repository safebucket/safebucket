import { Navigate, createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/settings/")({
  component: Settings,
});

function Settings() {
  return <Navigate to="/settings/profile" />;
}
