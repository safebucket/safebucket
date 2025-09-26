import React, { useEffect, useState } from "react";

import Cookies from "js-cookie";
import { jwtDecode } from "jwt-decode";
import { useForm } from "react-hook-form";
import type { SubmitHandler } from "react-hook-form";

import type {
  IJWTPayload,
  ILoginForm,
  ILoginResponse,
  IUser,
  Session,
  Status,
} from "@/components/auth-view/types/session";
import { SessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { router } from "@/main.tsx";
import { api, fetchApi } from "@/lib/api";
import { getApiUrl } from "@/lib/config.ts";

export const SessionProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
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

  const { register, handleSubmit, watch } = useForm<ILoginForm>();

  useEffect(() => {
    setStatus("loading");
    if (accessToken && authProvider) {
      const decoded = jwtDecode<IJWTPayload>(accessToken);

      fetchApi<IUser>(`/users/${decoded.user_id}`).then((res) =>
        setSession({
          loggedUser: res,
          accessToken: accessToken,
          refreshToken: refreshToken,
          authProvider: authProvider,
        }),
      );

      setStatus("authenticated");
    } else {
      setSession(null);
      setStatus("unauthenticated");
      if (!location.pathname.startsWith("/invites/")) {
        router.navigate({ to: "/auth/login" });
      }
    }
  }, [router, accessToken, refreshToken, authProvider]);

  const login = async (provider: string) => {
    setStatus("loading");
    const apiUrl = await getApiUrl();
    window.location.href = `${apiUrl}/auth/providers/${provider}/begin`;
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
      router.navigate({ to: "/auth/complete" });
    });
  };

  const logout = () => {
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
        watch,
        logout,
        session,
        status,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
};
