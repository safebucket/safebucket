import React, { createContext, useContext, useEffect, useState } from "react";

import Cookies from "js-cookie";
import { useRouter } from "next/navigation";

import { ISessionContext, Session, Status } from "@/app/auth/types/session";

const SessionContext = createContext<ISessionContext>({} as ISessionContext);

export const SessionProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const router = useRouter();
  const [accessToken, setAccessToken] = useState(
    Cookies.get("safebucket_access_token"),
  );
  const [refreshToken, _] = useState(Cookies.get("safebucket_refresh_token"));
  const [authProvider, setAuthProvider] = useState(
    Cookies.get("safebucket_auth_provider"),
  );
  const [session, setSession] = useState<Session | null>(null);
  const [status, setStatus] = useState<Status>("unauthenticated");

  useEffect(() => {
    setStatus("loading");
    if (accessToken && authProvider) {
      setSession({
        accessToken: accessToken,
        refreshToken: refreshToken,
        authProvider: authProvider,
      });
      setStatus("authenticated");
    } else {
      setSession(null);
      setStatus("unauthenticated");
      router.push("/auth/login");
    }
  }, [router, accessToken, refreshToken, authProvider]);

  const login = async (provider: string) => {
    setStatus("loading");
    router.push(`${process.env.NEXT_PUBLIC_API_URL}/auth/${provider}/begin`);
  };

  const logout = async () => {
    Cookies.remove("safebucket_access_token");
    setAccessToken(undefined);
    Cookies.remove("safebucket_auth_provider");
    setAuthProvider(undefined);
  };

  return (
    <SessionContext.Provider value={{ login, logout, session, status }}>
      {children}
    </SessionContext.Provider>
  );
};

export const useSession = () => {
  return useContext(SessionContext);
};
