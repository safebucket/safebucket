import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IPublicShareContent, IShare } from "@/types/share";
import { ShareLanding } from "@/components/share-view/components/ShareLanding.tsx";
import { SharePasswordForm } from "@/components/share-view/components/SharePasswordForm.tsx";
import { ShareContentView } from "@/components/share-view/components/ShareContentView.tsx";
import { mockConsumeShare } from "@/components/share-view/mock";

type PageState =
  | { step: "idle" }
  | { step: "password" }
  | { step: "content"; share: IShare; content: IPublicShareContent };

interface IShareConsumerPageProps {
  uuid: string;
}

export const ShareViewPage: FC<IShareConsumerPageProps> = ({ uuid }) => {
  const { t } = useTranslation();
  const [state, setState] = useState<PageState>({ step: "idle" });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const consumeShare = async (password?: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await mockConsumeShare(uuid, password);

      if (response.password_required) {
        setState({ step: "password" });
      } else {
        setState({
          step: "content",
          share: response.share,
          content: response.content,
        });
      }
    } catch (err) {
      const code = err instanceof Error ? err.message : "default";
      setError(t(`errors.${code}`));

      if (state.step === "idle") {
        setState({ step: "idle" });
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleAccess = () => consumeShare();

  const handlePasswordSubmit = (password: string) => consumeShare(password);

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
      return <ShareContentView share={state.share} content={state.content} />;
  }
};
