import { createFileRoute } from "@tanstack/react-router";
import { HomeView } from "@/components/home-view/HomeView.tsx";

export const Route = createFileRoute("/_authenticated/")({
  component: HomeView,
});
