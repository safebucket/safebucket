"use client";

import React from "react";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketSkeleton } from "@/components/bucket-view/components/BucketSkeleton";
import { useBucketData } from "@/components/bucket-view/hooks/useBucketData";

const files = [
  {
    id: 8,
    name: "projects",
    size: "6.7 MB",
    modified: "2023-08-15",
    type: "folder",
    files: [
      {
        id: 10,
        name: "Document.pdf",
        size: "2.3 MB",
        modified: "2023-04-15",
        type: "pdf",
      },
      {
        id: 111,
        name: "confidential",
        size: "2.3 MB",
        modified: "2023-04-15",
        type: "folder",
        files: [
          {
            id: 222,
            name: "secrets.txt",
            size: "0.3 MB",
            modified: "2023-04-15",
            type: "txt",
          },
        ],
      },
    ],
  },
  {
    id: 0,
    name: "Document.pdf",
    size: "2.3 MB",
    modified: "2023-04-15",
    type: "pdf",
  },
  {
    id: 1,
    name: "Presentation.pptx",
    size: "5.1 MB",
    modified: "2023-03-28",
    type: "pptx",
  },
  {
    id: 2,
    name: "Image.jpg",
    size: "1.7 MB",
    modified: "2023-05-02",
    type: "jpg",
  },
  {
    id: 3,
    name: "Spreadsheet.xlsx",
    size: "3.9 MB",
    modified: "2023-02-10",
    type: "xlsx",
  },
  {
    id: 4,
    name: "Video.mp4",
    size: "12.4 MB",
    modified: "2023-06-01",
    type: "mp4",
  },
  {
    id: 5,
    name: "Audio.mp3",
    size: "4.2 MB",
    modified: "2023-01-20",
    type: "mp3",
  },
  {
    id: 6,
    name: "Document2.pdf",
    size: "1.9 MB",
    modified: "2023-07-05",
    type: "pdf",
  },
  {
    id: 7,
    name: "Presentation2.pptx",
    size: "6.7 MB",
    modified: "2023-08-15",
    type: "pptx",
  },
];

interface IBucketProps {
  params: { id: string; path: string[] };
}

export default function Bucket({ params }: IBucketProps) {
  const { bucket, isLoading } = useBucketData(params.id);

  // FIXME: Remove when endpoint returns files
  if (!isLoading) bucket!.files = files;

  return (
    <div className="flex-1">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        {isLoading ? (
          <BucketSkeleton />
        ) : (
          <BucketView bucket={bucket!} path={params.path || []} />
        )}
      </div>
    </div>
  );
}
