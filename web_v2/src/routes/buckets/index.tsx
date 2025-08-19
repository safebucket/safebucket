import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/buckets/")({
  component: BucketsIndexComponent,
});

function BucketsIndexComponent() {
  // This could redirect to a buckets list or dashboard
  return (
    <div className="flex-1 w-full">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        <h1>Buckets</h1>
        <p>Select a bucket to view its contents.</p>
      </div>
    </div>
  );
}