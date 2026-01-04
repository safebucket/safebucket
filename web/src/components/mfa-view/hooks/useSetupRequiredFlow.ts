import { useState, useEffect, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";
import type {
  SetupRequiredViewMode,
  ISetupRequiredFlowState,
} from "../helpers/types";
import { useMFAAuth } from "@/context/MFAAuthContext";

export interface IUseSetupRequiredFlowReturn extends ISetupRequiredFlowState {
  userId: string | null;
  mfaToken: string | null;
}

export function useSetupRequiredFlow(): IUseSetupRequiredFlowReturn {
  const navigate = useNavigate();
  const { mfaToken: mfaTokenFromContext, userId: userIdFromContext } =
    useMFAAuth();
  const [viewMode, setViewMode] = useState<SetupRequiredViewMode>("loading");
  const [sessionError, setSessionError] = useState(false);
  const setupStarted = useRef(false);

  useEffect(() => {
    if (viewMode === "loading" && !setupStarted.current) {
      // Get user ID from MFA context (set during login when MFA setup is required)
      if (userIdFromContext && mfaTokenFromContext) {
        setupStarted.current = true;
        setViewMode("name");
      } else {
        setSessionError(true);
        navigate({ to: "/auth/login", search: { redirect: undefined } });
      }
    }
  }, [viewMode, navigate, userIdFromContext, mfaTokenFromContext]);

  const handleSuccess = () => {
    setViewMode("success");
  };

  const handleError = () => {
    setViewMode("error");
  };

  const handleRetry = () => {
    setViewMode("name");
  };

  return {
    viewMode,
    userId: userIdFromContext,
    mfaToken: mfaTokenFromContext,
    sessionError,
    handleSuccess,
    handleError,
    handleRetry,
  };
}
