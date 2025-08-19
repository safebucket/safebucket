export function ActivityViewSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 5 }).map((_, index) => (
        <div key={index} className="flex items-start space-x-4 py-4">
          <div className="h-10 w-10 animate-pulse rounded-full bg-gray-200" />
          <div className="flex-1 space-y-2">
            <div className="flex items-center space-x-2">
              <div className="h-4 w-32 animate-pulse rounded bg-gray-200" />
              <div className="h-6 w-6 animate-pulse rounded-full bg-gray-200" />
            </div>
            <div className="h-3 w-full animate-pulse rounded bg-gray-200" />
            <div className="h-3 w-24 animate-pulse rounded bg-gray-200" />
          </div>
        </div>
      ))}
    </div>
  );
}
