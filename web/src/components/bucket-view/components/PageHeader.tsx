import type { FC, ReactNode } from "react";

interface IPageHeaderProps {
  title: string;
  actions?: ReactNode;
}

export const PageHeader: FC<IPageHeaderProps> = ({
  title,
  actions,
}: IPageHeaderProps) => {
  return (
    <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <h2 className="min-w-0 truncate text-2xl font-semibold tracking-tight">
        {title}
      </h2>
      {actions ? (
        <div className="flex shrink-0 items-center gap-2">{actions}</div>
      ) : null}
    </div>
  );
};
