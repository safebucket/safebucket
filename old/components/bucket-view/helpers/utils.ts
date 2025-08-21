import { IFile } from "@/components/bucket-view/helpers/types";

export const getFileType = (extension: string): string => {
  switch (extension.toLowerCase()) {
    // Text files
    case "txt":
    case "md":
    case "html":
    case "xml":
    case "json":
    case "csv":
      return "text";

    // Image files
    case "jpg":
    case "jpeg":
    case "png":
    case "gif":
    case "bmp":
    case "svg":
    case "webp":
      return "image";

    // Audio files
    case "mp3":
    case "wav":
    case "ogg":
    case "flac":
    case "aac":
      return "audio";

    // Video files
    case "mp4":
    case "avi":
    case "mkv":
    case "mov":
    case "webm":
      return "video";

    // Document files
    case "pdf":
    case "doc":
    case "docx":
    case "ppt":
    case "pptx":
    case "xls":
    case "xlsx":
      return "document";

    // Compressed files
    case "zip":
    case "rar":
    case "7z":
    case "tar":
    case "gz":
      return "archive";

    // Executable files
    case "exe":
    case "sh":
    case "bat":
    case "jar":
      return "executable";

    case "folder":
      return "folder";

    default:
      return "unknown";
  }
};

const buildFileStructure = (files: IFile[]) => {
  const map: Record<string, IFile[]> = {};

  files.forEach((file) => {
    (map[file.path] ??= []).push(file);
  });

  return map;
};

export const filesToShow = (files: IFile[], path: string) => {
  const fileStructure = buildFileStructure(files);
  return fileStructure[path] || [];
};
