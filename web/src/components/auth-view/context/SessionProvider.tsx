import { useReducer, useEffect, useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";

import type { Session, Status } from "@/components/auth-view/types/session";
import { SessionContext } from "@/components/auth-view/hooks/useSessionContext";
import {
  getCurrentSession,
  loginWithProvider,
  logout as authLogout,
  setAuthenticationState as authSetAuthenticationState,
} from "@/lib/auth-service";

// ============================================================================
// State Management with useReducer
// ============================================================================

interface SessionState {
  session: Session | null;
  status: Status;
}

type SessionAction =
  | { type: "SET_LOADING" }
  | { type: "SET_AUTHENTICATED"; payload: Session }
  | { type: "SET_UNAUTHENTICATED" }
  | { type: "REFRESH_SESSION" };

const initialState: SessionState = {
  session: null,
  status: "loading",
};

function sessionReducer(
  state: SessionState,
  action: SessionAction,
): SessionState {
  switch (action.type) {
    case "SET_LOADING":
      return {
        ...state,
        status: "loading",
      };

    case "SET_AUTHENTICATED":
      return {
        session: action.payload,
        status: "authenticated",
      };

    case "SET_UNAUTHENTICATED":
      return {
        session: null,
        status: "unauthenticated",
      };

    case "REFRESH_SESSION": {
      const session = getCurrentSession();
      if (session) {
        return {
          session,
          status: "authenticated",
        };
      }
      return {
        session: null,
        status: "unauthenticated",
      };
    }

    default:
      return state;
  }
}

// ============================================================================
// SessionProvider Component
// ============================================================================

interface SessionProviderProps {
  children: React.ReactNode;
}

export const SessionProvider = ({ children }: SessionProviderProps) => {
  const [state, dispatch] = useReducer(sessionReducer, initialState);
  const navigate = useNavigate();

  // Initialize session from cookies on mount
  useEffect(() => {
    const session = getCurrentSession();

    if (session) {
      dispatch({ type: "SET_AUTHENTICATED", payload: session });
    } else {
      dispatch({ type: "SET_UNAUTHENTICATED" });
    }
  }, []);

  // Refresh session when cookies change
  const refreshSession = useCallback(() => {
    dispatch({ type: "REFRESH_SESSION" });
  }, []);

  // OAuth provider login
  const login = useCallback((provider: string) => {
    dispatch({ type: "SET_LOADING" });
    loginWithProvider(provider);
  }, []);

  // Logout function
  const logout = useCallback(() => {
    dispatch({ type: "SET_LOADING" });

    // Clear cookies and state
    authLogout();
    dispatch({ type: "SET_UNAUTHENTICATED" });

    // Navigate to login
    navigate({ to: "/auth/login", search: { redirect: undefined } });
  }, [navigate]);

  // Manually set authentication state (for password reset, invite acceptance)
  const setAuthenticationState = useCallback(
    (accessToken: string, refreshToken: string, provider: string) => {
      // Set cookies via auth service
      authSetAuthenticationState(accessToken, refreshToken, provider);

      // Refresh session from cookies
      const session = getCurrentSession();
      if (session) {
        dispatch({ type: "SET_AUTHENTICATED", payload: session });
      }
    },
    [],
  );

  return (
    <SessionContext.Provider
      value={{
        session: state.session,
        status: state.status,
        login,
        logout,
        setAuthenticationState,
        refreshSession,
      }}
    >
      {children}
    </SessionContext.Provider>
  );
};
