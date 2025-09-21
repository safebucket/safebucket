import type { IFile } from "@/types/file.ts";

export interface Invites {
  email: string;
  group: string;
}

export interface IBucket {
  id: string;
  name: string;
  files: Array<IFile>;
  created_by: string;
  created_at: string;
  updated_at: string;
}
