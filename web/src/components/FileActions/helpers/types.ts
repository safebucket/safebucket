export interface IFileActions {
  createFolder: (name: string) => void;
  deleteFile: (fileId: string, filename: string, isFolder?: boolean) => void;
  downloadFile: (fileId: string, filename: string) => void;
}
