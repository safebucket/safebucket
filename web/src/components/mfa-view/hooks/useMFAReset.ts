import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import type {
  IMFAResetRequestResponse,
  ResetStep,
} from "@/components/mfa-view/helpers/types";
import { api } from "@/lib/api";
import { translateError } from "@/lib/errors";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { MFA_CODE_LENGTH } from "@/components/mfa-view/helpers/constants";

export interface UseMFAResetReturn {
  step: ResetStep;
  password: string;
  setPassword: (password: string) => void;
  code: string;
  setCode: (code: string) => void;
  error: string | null;
  isLoading: boolean;
  requestReset: () => Promise<void>;
  verifyReset: () => Promise<void>;
  reset: () => void;
}

export function useMFAReset(userId: string): UseMFAResetReturn {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [step, setStep] = useState<ResetStep>("password");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [challengeId, setChallengeId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const requestResetMutation = useMutation({
    mutationFn: (password: string) =>
      api.post<IMFAResetRequestResponse>(`/users/${userId}/mfa/reset`, {
        password,
      }),
    onError: (error: Error) => errorToast(error),
  });

  const verifyResetMutation = useMutation({
    mutationFn: ({
      challengeId,
      code,
    }: {
      challengeId: string;
      code: string;
    }) => api.post(`/users/${userId}/mfa/reset/${challengeId}`, { code }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
      successToast("MFA has been reset successfully");
    },
    onError: (error: Error) => errorToast(error),
  });

  const reset = useCallback(() => {
    setStep("password");
    setPassword("");
    setCode("");
    setChallengeId(null);
    setError(null);
  }, []);

  const requestReset = useCallback(async () => {
    if (!password) {
      setError(t("auth.mfa.error_password_required"));
      return;
    }

    setError(null);
    try {
      const response = await requestResetMutation.mutateAsync(password);
      setChallengeId(response.challenge_id);
      setStep("email_sent");
    } catch (err) {
      setError(translateError(err, "auth.mfa.reset_request_error", t));
    }
  }, [password, requestResetMutation, t]);

  const verifyReset = useCallback(async () => {
    if (code.length !== MFA_CODE_LENGTH) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    if (!challengeId) {
      setError(t("auth.mfa.reset_request_error"));
      return;
    }

    setError(null);
    try {
      await verifyResetMutation.mutateAsync({ challengeId, code });
      setStep("success");
    } catch (err) {
      setError(translateError(err, "auth.mfa.reset_verify_error", t));
    }
  }, [code, challengeId, verifyResetMutation, t]);

  return {
    step,
    password,
    setPassword,
    code,
    setCode,
    error,
    isLoading: requestResetMutation.isPending || verifyResetMutation.isPending,
    requestReset,
    verifyReset,
    reset,
  };
}
