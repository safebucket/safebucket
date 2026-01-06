import { createFileRoute } from "@tanstack/react-router";
import { AdminBucketsView } from "@/components/admin-buckets/AdminBucketsView";

export const Route = createFileRoute("/_authenticated/admin/buckets/")({
  component: AdminBucketsView,
});
