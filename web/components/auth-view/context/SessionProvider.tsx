import React, { useEffect, useState } from "react";

import { api } from "@/lib/api";
import Cookies from "js-cookie";
import { useRouter } from "next/navigation";
import { SubmitHandler, useForm } from "react-hook-form";

import { SessionContext } from "@/components/auth-view/hooks/useSessionContext";
import {
  ILoginForm,
  ILoginResponse,
  Session,
  Status,
} from "@/components/auth-view/types/session";

export const SessionProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const router = useRouter();
  const [accessToken, setAccessToken] = useState(
    Cookies.get("safebucket_access_token"),
  );
  const [refreshToken, setRefreshToken] = useState(
    Cookies.get("safebucket_refresh_token"),
  );
  const [authProvider, setAuthProvider] = useState(
    Cookies.get("safebucket_auth_provider"),
  );
  const [session, setSession] = useState<Session | null>(null);
  const [status, setStatus] = useState<Status>("unauthenticated");

  const { register, handleSubmit } = useForm<ILoginForm>();

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
    router.push(
      `${process.env.NEXT_PUBLIC_API_URL}/auth/providers/${provider}/begin`,
    );
  };

  const localLogin: SubmitHandler<ILoginForm> = async (body) => {
    setStatus("loading");
    await api.post<ILoginResponse>("/auth/login", body).then((res) => {
      Cookies.set("safebucket_access_token", res.access_token);
      setAccessToken(res.access_token);
      Cookies.set("safebucket_refresh_token", res.refresh_token);
      setRefreshToken(res.refresh_token);
      Cookies.set("safebucket_auth_provider", "local");
      setAuthProvider("local");
      router.push("/auth/complete");
    });
  };

  const logout = async () => {
    Cookies.remove("safebucket_access_token");
    setAccessToken(undefined);
    Cookies.remove("safebucket_auth_provider");
    setAuthProvider(undefined);
  };

  return (
    <SessionContext.Provider
      value={{
        login,
        localLogin,
        register,
        handleSubmit,
        logout,
        session,
        status,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
};
