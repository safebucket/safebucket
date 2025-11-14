import type { Session } from "@/components/auth-view/types/session";
import { getCurrentSession } from "@/lib/auth-service";

/**
 * Auth Manager - Synchronous session initialization and management
 * No React dependencies - can be used before React renders
 */

/**
 * Initialize session synchronously from cookies
 * Called before router creation to avoid race conditions
 */
export function initializeSession(): Session | null {
  try {
    return getCurrentSession();
  } catch (error) {
    console.error("[Auth Manager] Failed to initialize session:", error);
    return null;
  }
}
