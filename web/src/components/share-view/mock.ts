import type { FileStatus, IFile } from "@/types/file";
import type { IFolder } from "@/types/folder";
import type { IConsumeShareResponse, IShare } from "@/types/share";

const mockFiles: Array<IFile> = [
  {
    id: "f1",
    name: "presentation-q4.pdf",
    size: 2_450_000,
    extension: "pdf",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-10T14:30:00Z",
    deleted_at: null,
    expires_at: null,
  },
  {
    id: "f2",
    name: "budget-2026.xlsx",
    size: 890_000,
    extension: "xlsx",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-11T09:15:00Z",
    deleted_at: null,
    expires_at: null,
  },
  {
    id: "f3",
    name: "team-photo.jpg",
    size: 4_200_000,
    extension: "jpg",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-12T16:45:00Z",
    deleted_at: null,
    expires_at: null,
  },
  {
    id: "f4",
    name: "meeting-notes.md",
    size: 12_000,
    extension: "md",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-13T11:00:00Z",
    deleted_at: null,
    expires_at: null,
  },
  {
    id: "f5",
    name: "demo-video.mp4",
    size: 45_000_000,
    extension: "mp4",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-14T08:20:00Z",
    deleted_at: null,
    expires_at: null,
  },
];

const mockFolderFiles: Array<IFile> = [
  {
    id: "ff1",
    name: "design-system.fig",
    size: 8_500_000,
    extension: "fig",
    folder_id: "folder1",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-09T10:00:00Z",
    deleted_at: null,
    expires_at: null,
  },
  {
    id: "ff2",
    name: "wireframes.png",
    size: 1_200_000,
    extension: "png",
    folder_id: "folder1",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-09T10:30:00Z",
    deleted_at: null,
    expires_at: null,
  },
  {
    id: "ff3",
    name: "readme.md",
    size: 5_000,
    extension: "md",
    folder_id: "subfolder1",
    status: "uploaded" as FileStatus,
    created_at: "2026-03-10T12:00:00Z",
    deleted_at: null,
    expires_at: null,
  },
];

const mockFolders: Array<IFolder> = [
  {
    id: "subfolder1",
    name: "Documentation",
    folder_id: "folder1",
    bucket_id: "b1",
    status: null,
    created_at: "2026-03-08T09:00:00Z",
    updated_at: "2026-03-08T09:00:00Z",
  },
];

const mockBucketFiles: Array<IFile> = [
  ...mockFiles.slice(0, 3),
  ...mockFolderFiles,
];

const mockBucketFolders: Array<IFolder> = [
  {
    id: "folder1",
    name: "Design Assets",
    bucket_id: "b1",
    status: null,
    created_at: "2026-03-07T09:00:00Z",
    updated_at: "2026-03-07T09:00:00Z",
  },
  ...mockFolders,
];

const mockShares: Record<
  string,
  {
    share: IShare;
    files: Array<IFile>;
    folders: Array<IFolder>;
    password?: string;
  }
> = {
  "test-uuid": {
    share: {
      id: "test-uuid",
      name: "Q4 Reports",
      bucket_id: "b1",
      folder_id: null,
      type: "files",
      expires_at: "2026-04-15T00:00:00Z",
      max_views: 10,
      current_views: 3,
      password_protected: false,
      allow_upload: false,
      max_uploads: null,
      current_uploads: 0,
      max_upload_size: null,
      files: [
        { id: "sf1", share_id: "test-uuid", file_id: "f1" },
        { id: "sf2", share_id: "test-uuid", file_id: "f2" },
        { id: "sf3", share_id: "test-uuid", file_id: "f4" },
      ],
      created_by: "user1",
      created_at: "2026-03-10T10:00:00Z",
    },
    files: [mockFiles[0], mockFiles[1], mockFiles[3]],
    folders: [],
  },
  "password-uuid": {
    share: {
      id: "password-uuid",
      name: "Project Alpha",
      bucket_id: "b1",
      folder_id: null,
      type: "bucket",
      expires_at: null,
      max_views: null,
      current_views: 7,
      password_protected: true,
      allow_upload: true,
      max_uploads: 5,
      current_uploads: 2,
      max_upload_size: 104_857_600,
      files: null,
      created_by: "user1",
      created_at: "2026-03-08T10:00:00Z",
    },
    files: mockBucketFiles,
    folders: mockBucketFolders,
    password: "password123",
  },
  "folder-uuid": {
    share: {
      id: "folder-uuid",
      name: "Design Assets",
      bucket_id: "b1",
      folder_id: "folder1",
      type: "folder",
      expires_at: "2026-05-01T00:00:00Z",
      max_views: 50,
      current_views: 12,
      password_protected: false,
      allow_upload: true,
      max_uploads: 10,
      current_uploads: 0,
      max_upload_size: 52_428_800,
      files: null,
      created_by: "user1",
      created_at: "2026-03-09T10:00:00Z",
    },
    files: mockFolderFiles,
    folders: mockFolders,
  },
};

export async function mockConsumeShare(
  uuid: string,
  password?: string,
): Promise<IConsumeShareResponse> {
  await new Promise((resolve) => setTimeout(resolve, 500));

  const entry = mockShares[uuid];
  if (!entry) {
    throw new Error("NOT_FOUND");
  }

  if (entry.share.password_protected && !password) {
    return { password_required: true };
  }

  if (entry.share.password_protected && password !== entry.password) {
    throw new Error("INVALID_PASSWORD");
  }

  return {
    password_required: false,
    share: entry.share,
    content: {
      files: entry.files,
      folders: entry.folders,
    },
  };
}
