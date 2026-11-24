import type { IUserInfo } from "@/types/user";

export interface IFileComment {
  id: string;
  body: string;
  user?: IUserInfo;
  version: number;
  created_at: string;
  updated_at: string;
}
