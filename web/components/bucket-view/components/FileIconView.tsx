import React, { FC } from "react";

import {
  FileAudio,
  FileChartColumn,
  FileChartPie,
  FileCode,
  FileIcon,
  FileImage,
  FileTerminal,
  FileText,
  FileVideo,
  FolderClosed,
} from "lucide-react";

interface IFileIconViewProps {
  extension: string;
  className: string;
}

export const FileIconView: FC<IFileIconViewProps> = ({
  extension,
  className,
}: IFileIconViewProps) => {
  switch (extension) {
    case "txt":
    case "md":
    case "pdf":
      return <FileText className={className} />;

    case "xls":
    case "xlsx":
    case "numbers":
      return <FileChartColumn className={className} />;

    case "ppt":
    case "pptx":
    case "pages":
      return <FileChartPie className={className} />;

    case "jpg":
    case "jpeg":
    case "png":
    case "gif":
    case "bmp":
    case "svg":
    case "webp":
      return <FileImage className={className} />;

    case "mp3":
    case "wav":
    case "ogg":
    case "flac":
    case "aac":
      return <FileAudio className={className} />;

    case "mp4":
    case "avi":
    case "mkv":
    case "mov":
    case "webm":
      return <FileVideo className={className} />;

    case "py":
    case "ts":
    case "tsx":
    case "java":
      return <FileCode className={className} />;

    case "exe":
    case "sh":
    case "bat":
    case "jar":
      return <FileTerminal className={className} />;

    case "folder":
      return <FolderClosed className={className} />;

    default:
      return <FileIcon className={className} />;
  }
};
