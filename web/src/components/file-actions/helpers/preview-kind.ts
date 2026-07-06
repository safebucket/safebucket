export type PreviewKind =
  "image" | "video" | "audio" | "pdf" | "text" | "unsupported";

const IMAGE = new Set([
  "png",
  "jpg",
  "jpeg",
  "gif",
  "webp",
  "svg",
  "bmp",
  "ico",
  "avif",
]);
const VIDEO = new Set(["mp4", "webm", "ogv", "mov", "m4v"]);
const AUDIO = new Set(["mp3", "wav", "ogg", "m4a", "flac", "aac"]);
const TEXT = new Set([
  "txt",
  "log",
  "md",
  "csv",
  "tsv",
  "json",
  "xml",
  "yaml",
  "yml",
  "html",
  "htm",
  "css",
  "js",
  "ts",
  "tsx",
  "jsx",
  "go",
  "py",
  "rs",
  "rb",
  "java",
  "kt",
  "c",
  "h",
  "cpp",
  "sh",
  "sql",
  "toml",
  "ini",
  "env",
]);

export const getPreviewKind = (extension: string): PreviewKind => {
  const ext = extension.toLowerCase();
  if (IMAGE.has(ext)) return "image";
  if (VIDEO.has(ext)) return "video";
  if (AUDIO.has(ext)) return "audio";
  if (ext === "pdf") return "pdf";
  if (TEXT.has(ext)) return "text";
  return "unsupported";
};
