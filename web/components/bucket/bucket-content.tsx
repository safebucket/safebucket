import React from "react";

import { FileView } from "@/components/fileview";

export function BucketContent({ files }) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
      {files.map((file) => (
        <FileView key={file.id} file={file} />
      ))}
    </div>
  );
}
