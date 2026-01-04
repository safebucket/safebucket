import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  type ReactNode,
} from "react";
import type { IMFADevice } from "@/components/mfa-view/helpers/types";

interface IMFAAuthContext {
  mfaToken: string | null;
  userId: string | null;
  devices: IMFADevice[];
  setMFAAuth: (token: string, userId: string, devices: IMFADevice[]) => void;
  clearMFAAuth: () => void;
}

const MFAAuthContext = createContext<IMFAAuthContext | null>(null);

export function MFAAuthProvider({ children }: { children: ReactNode }) {
  const [mfaToken, setMfaToken] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);
  const [devices, setDevices] = useState<IMFADevice[]>([]);

  const setMFAAuth = useCallback(
    (token: string, uid: string, deviceList: IMFADevice[]) => {
      setMfaToken(token);
      setUserId(uid);
      setDevices(deviceList);
    },
    [],
  );

  const clearMFAAuth = useCallback(() => {
    setMfaToken(null);
    setUserId(null);
    setDevices([]);
  }, []);

  useEffect(() => {
    if (mfaToken) {
      const timeout = setTimeout(
        () => {
          clearMFAAuth();
        },
        15 * 60 * 1000,
      );

      return () => clearTimeout(timeout);
    }
  }, [mfaToken, clearMFAAuth]);

  return (
    <MFAAuthContext.Provider
      value={{ mfaToken, userId, devices, setMFAAuth, clearMFAAuth }}
    >
      {children}
    </MFAAuthContext.Provider>
  );
}

export function useMFAAuth(): IMFAAuthContext {
  const context = useContext(MFAAuthContext);
  if (!context) {
    throw new Error("useMFAAuth must be used within MFAAuthProvider");
  }
  return context;
}
