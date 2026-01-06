import type { TFunction } from "i18next";

/**
 * Translates an error by extracting the error code from the backend response
 * and looking it up in the i18n errors namespace.
 *
 * @param error - The error object from the backend
 * @param fallbackKey - The translation key to use if the error code is not found
 * @param t - The i18next translation function
 * @returns The translated error message
 *
 * @example
 * ```ts
 * try {
 *   await api.post('/endpoint');
 * } catch (err) {
 *   const message = translateError(err, 'errors.default', t);
 *   setError(message);
 * }
 * ```
 */
export function translateError(
  error: unknown,
  fallbackKey: string,
  t: TFunction,
): string {
  const errorMessage = error instanceof Error ? error.message : "";
  return t(`errors.${errorMessage}`, { defaultValue: t(fallbackKey) });
}
