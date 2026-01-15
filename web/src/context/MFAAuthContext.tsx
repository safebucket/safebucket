import {
  
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState
} from "react";
import type {ReactNode} from "react";
import type {
  IMFADevice,
  IMFADevicesResponse,
} from "@/components/mfa-view/helpers/types";
import { fetchApi } from "@/lib/api";

interface IMFAAuthContext {
  restrictedToken: string | null;
  userId: string | null;
  devices: Array<IMFADevice>;
  isLoadingDevices: boolean;
  setMFAAuth: (token: string, userId: string) => void;
  clearMFAAuth: () => void;
}

const MFAAuthContext = createContext<IMFAAuthContext | null>(null);

export function MFAAuthProvider({ children }: { children: ReactNode }) {
  const [restrictedToken, setRestrictedToken] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);
  const [devices, setDevices] = useState<Array<IMFADevice>>([]);
  const [isLoadingDevices, setIsLoadingDevices] = useState(false);
  const [devicesFetched, setDevicesFetched] = useState(false);

  const setMFAAuth = useCallback((token: string, uid: string) => {
    setRestrictedToken(token);
    setUserId(uid);
    setDevices([]);
    setDevicesFetched(false);
  }, []);

  const clearMFAAuth = useCallback(() => {
    setRestrictedToken(null);
    setUserId(null);
    setDevices([]);
    setDevicesFetched(false);
  }, []);

  // Fetch devices when token and userId are set
  useEffect(() => {
    if (restrictedToken && userId && !devicesFetched && !isLoadingDevices) {
      const fetchDevices = async () => {
        setIsLoadingDevices(true);
        try {
          const response = await fetchApi<IMFADevicesResponse>(
            `/users/${userId}/mfa/devices`,
            { headers: { Authorization: `Bearer ${restrictedToken}` } },
          );
          setDevices(response.devices);
        } catch {
          // If fetch fails, keep devices empty (will trigger setup flow)
          setDevices([]);
        } finally {
          setIsLoadingDevices(false);
          setDevicesFetched(true);
        }
      };
      fetchDevices();
    }
  }, [restrictedToken, userId, devicesFetched, isLoadingDevices]);

  // Auto-clear after 15 minutes
  useEffect(() => {
    if (restrictedToken) {
      const timeout = setTimeout(
        () => {
          clearMFAAuth();
        },
        15 * 60 * 1000,
      );

      return () => clearTimeout(timeout);
    }
  }, [restrictedToken, clearMFAAuth]);

  return (
    <MFAAuthContext.Provider
      value={{
        restrictedToken,
        userId,
        devices,
        isLoadingDevices,
        setMFAAuth,
        clearMFAAuth,
      }}
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
