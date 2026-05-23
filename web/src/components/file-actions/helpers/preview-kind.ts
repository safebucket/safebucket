export type PreviewKind =
  | "image"
  | "video"
  | "audio"
  | "pdf"
  | "text"
  | "unsupported";

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

const TEXT_MIME: Record<string, string> = {
  json: "application/json",
  xml: "application/xml",
  html: "text/html",
  htm: "text/html",
  css: "text/css",
  csv: "text/csv",
  tsv: "text/tab-separated-values",
  md: "text/markdown",
};

export const getPreviewMime = (
  kind: PreviewKind,
  extension: string,
): string => {
  if (kind === "pdf") return "application/pdf";
  if (kind === "text") {
    const ext = extension.toLowerCase();
    return TEXT_MIME[ext] ?? "text/plain";
  }
  return "application/octet-stream";
};
