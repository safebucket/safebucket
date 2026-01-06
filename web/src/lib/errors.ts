import type { TFunction } from "i18next";

export function translateError(
  error: unknown,
  fallbackKey: string,
  t: TFunction,
): string {
  const errorMessage = error instanceof Error ? error.message : "";
  return t(`errors.${errorMessage}`, { defaultValue: t(fallbackKey) });
}
