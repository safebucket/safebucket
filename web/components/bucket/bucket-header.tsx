import React, { useState } from "react";

import ShareFileDialog from "@/components/sharefile";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function BucketHeader({ bucketName }) {
  const [sortBy, setSortBy] = useState("name");
  const [sortOrder, setSortOrder] = useState("asc");
  const [filterType, setFilterType] = useState("all");

  const handleSort = (field) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(field);
      setSortOrder("asc");
    }
  };
  const handleFilter = (type) => {
    setFilterType(type);
  };

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{bucketName}</h1>
        <div className="flex items-center gap-4">
          <Select value={filterType} onValueChange={handleFilter}>
            <SelectTrigger>
              <SelectValue placeholder="Filter by type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="pdf">PDF</SelectItem>
              <SelectItem value="pptx">PowerPoint</SelectItem>
              <SelectItem value="jpg">Image</SelectItem>
              <SelectItem value="xlsx">Excel</SelectItem>
              <SelectItem value="mp4">Video</SelectItem>
              <SelectItem value="mp3">Audio</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            onClick={() => handleSort("name")}
            className={sortBy === "name" ? "font-medium" : ""}
          >
            Name{" "}
            {sortBy === "name" && (sortOrder === "asc" ? "\u2191" : "\u2193")}
          </Button>
          <Button
            variant="outline"
            onClick={() => handleSort("size")}
            className={sortBy === "size" ? "font-medium" : ""}
          >
            Size{" "}
            {sortBy === "size" && (sortOrder === "asc" ? "\u2191" : "\u2193")}
          </Button>
          <Button
            variant="outline"
            onClick={() => handleSort("modified")}
            className={sortBy === "modified" ? "font-medium" : ""}
          >
            Modified{" "}
            {sortBy === "modified" &&
              (sortOrder === "asc" ? "\u2191" : "\u2193")}
          </Button>
          <ShareFileDialog />
        </div>
      </div>
    </div>
  );
}
