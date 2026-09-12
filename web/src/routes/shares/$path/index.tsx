import { createFileRoute } from "@tanstack/react-router";
import { ShareViewPage } from "@/components/share-view/ShareViewPage.tsx";

export const Route = createFileRoute("/shares/$path/")({
  component: SharePage,
});

function SharePage() {
  const { path } = Route.useParams();

  return <ShareViewPage path={path} />;
}
