export interface IUserDisplayInfo {
  first_name?: string;
  last_name?: string;
  email: string;
}

export interface IUserInfo extends IUserDisplayInfo {
  id: string;
  first_name: string;
  last_name: string;
}

export function getUserDisplayName(
  user: IUserDisplayInfo | undefined,
  fallback: string,
): string {
  if (!user) return fallback;

  const name = [user.first_name, user.last_name].filter(Boolean).join(" ");
  return name || user.email;
}
