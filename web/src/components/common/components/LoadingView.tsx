import { LoaderCircle } from "lucide-react";
import type { FC } from "react";

export const LoadingView: FC = () => {
  return (
    <div className="flex h-screen w-screen items-center justify-center text-center">
      <LoaderCircle className="m-2 animate-spin" />
      Loading...
    </div>
  );
};
