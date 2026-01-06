import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";

export interface IBucket {
  id: string;
  name: string;
  files: Array<IFile>;
  folders: Array<IFolder>;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export type BucketItem = IFile | IFolder;

export type BucketGroup = "owner" | "contributor" | "viewer";

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
