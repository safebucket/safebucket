import type { FC, ReactNode } from "react";

interface IPageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
}

export const PageHeader: FC<IPageHeaderProps> = ({
  title,
  description,
  actions,
}: IPageHeaderProps) => {
  return (
    <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex min-w-0 flex-col gap-1">
        <h2 className="min-w-0 truncate text-2xl font-semibold tracking-tight">
          {title}
        </h2>
        {description ? (
          <p className="text-muted-foreground line-clamp-2 text-sm wrap-break-word">
            {description}
          </p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex shrink-0 items-center gap-2">{actions}</div>
      ) : null}
    </div>
  );
};
