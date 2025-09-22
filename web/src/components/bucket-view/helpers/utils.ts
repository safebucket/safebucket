import type { IFile } from "@/types/file.ts";

const buildFileStructure = (files: Array<IFile>) => {
  const map: Record<string, Array<IFile>> = {};

  files.forEach((file) => {
    (map[file.path] ??= []).push(file);
  });

  return map;
};

export const filesToShow = (files: Array<IFile>, path: string) => {
  const fileStructure = buildFileStructure(files);
  return fileStructure[path] || [];
};
