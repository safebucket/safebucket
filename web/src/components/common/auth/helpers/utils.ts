import type { IProvider } from "@/types/auth_providers.ts";

export const checkEmailDomain = (
  email: string,
  providers: Array<IProvider>,
) => {
  if (!email.includes("@")) return null;
  const emailDomain = email.split("@")[1].toLowerCase();

  for (const provider of providers) {
    for (const domain of provider.domains) {
      if (domain.toLowerCase() === emailDomain) {
        return provider;
      }
    }
  }
  return null;
};
