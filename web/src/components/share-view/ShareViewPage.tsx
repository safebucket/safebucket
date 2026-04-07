import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";

import type { IPublicShareResponse } from "@/types/share";
import { ShareLanding } from "@/components/share-view/components/ShareLanding.tsx";
import { SharePasswordForm } from "@/components/share-view/components/SharePasswordForm.tsx";
import { ShareContentView } from "@/components/share-view/components/ShareContentView.tsx";
import {
  shareContentQueryOptions,
  useShareAuthMutation,
} from "@/queries/share";

type PageState =
  | { step: "idle" }
  | { step: "password" }
  | { step: "content"; shareContent: IPublicShareResponse };

interface IShareConsumerPageProps {
  uuid: string;
}

export const ShareViewPage: FC<IShareConsumerPageProps> = ({ uuid }) => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [state, setState] = useState<PageState>({ step: "idle" });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shareToken, setShareToken] = useState<string | null>(null);

  const authMutation = useShareAuthMutation();

  const fetchShareContent = async (token: string | null) => {
    const data = await queryClient.fetchQuery(
      shareContentQueryOptions(uuid, token),
    );
    setState({ step: "content", shareContent: data });
  };

  const handleAccess = async () => {
    setIsLoading(true);
    setError(null);

    try {
      await fetchShareContent(null);
    } catch (err) {
      const code = err instanceof Error ? err.message : "INTERNAL_SERVER_ERROR";
      if (code === "SHARE_TOKEN_REQUIRED") {
        setState({ step: "password" });
      } else {
        setError(t(`errors.${code}`));
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handlePasswordSubmit = async (password: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const { token } = await authMutation.mutateAsync({
        shareId: uuid,
        password,
      });
      setShareToken(token);
      await fetchShareContent(token);
    } catch (err) {
      const code = err instanceof Error ? err.message : "INTERNAL_SERVER_ERROR";
      setError(t(`errors.${code}`));
    } finally {
      setIsLoading(false);
    }
  };

  const handleBack = () => {
    setState({ step: "idle" });
    setError(null);
  };

  switch (state.step) {
    case "idle":
      return (
        <ShareLanding
          onAccess={handleAccess}
          isLoading={isLoading}
          error={error}
        />
      );
    case "password":
      return (
        <SharePasswordForm
          onSubmit={handlePasswordSubmit}
          onBack={handleBack}
          isLoading={isLoading}
          error={error}
        />
      );
    case "content":
      return (
        <ShareContentView
          shareId={uuid}
          shareContent={state.shareContent}
          token={shareToken}
        />
      );
  }
};
