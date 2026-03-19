import { createFileRoute } from "@tanstack/react-router";
import { ShareViewPage } from "@/components/share-view/ShareViewPage.tsx";

export const Route = createFileRoute("/shares/$uuid/")({
  component: SharePage,
});

function SharePage() {
  const { uuid } = Route.useParams();

  return <ShareViewPage uuid={uuid} />;
}
