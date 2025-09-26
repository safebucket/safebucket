import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IProvider } from "@/types/auth_providers.ts";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext.ts";
import { Button } from "@/components/ui/button.tsx";

interface IAuthProvidersButtonsProps {
  providers: Array<IProvider>;
}

export const AuthProvidersButtons: FC<IAuthProvidersButtonsProps> = ({
  providers,
}) => {
  const { login } = useSessionContext();
  const { t } = useTranslation();

  return (
    <div
      className={`grid ${providers.length === 1 ? "grid-cols-1" : "grid-cols-2"}`}
    >
      {providers.map((provider) => (
        <Button
          key={provider.id}
          variant="outline"
          onClick={() => login(provider.id)}
        >
          <img
            width={15}
            height={15}
            alt={`${provider.name} logo`}
            src={`/${provider.id}.svg`}
            onError={(e) => {
              const target = e.target as HTMLImageElement;
              target.src = "/login.svg";
            }}
            className="mr-2 h-4 w-4"
          />
          {t("auth.continue_with", { name: provider.name })}
        </Button>
      ))}
    </div>
  );
};
