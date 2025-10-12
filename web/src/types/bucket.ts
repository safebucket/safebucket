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

export const bucketGroups = [
  { id: "viewer", name: "Viewer", description: "Can view and download files" },
  {
    id: "contributor",
    name: "Contributor",
    description: "Can view, download and upload files",
  },
  {
    id: "owner",
    name: "Owner",
    description: "Can manage files and update the bucket",
  },
];

export const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
