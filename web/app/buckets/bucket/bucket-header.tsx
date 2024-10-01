import React, { FC, useState } from "react";

import { PlusCircle } from "lucide-react";

import { Bucket } from "@/app/buckets/helpers/types";

import { CustomDialog } from "@/components/custom-dialog";
import { DatePickerDemo } from "@/components/datepicker";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";

interface IBucketHeaderProps {
  bucket: Bucket|undefined;
  isLoading: boolean;
}

export const BucketHeader: FC<IBucketHeaderProps> = ({
  bucket,
  isLoading,
}: IBucketHeaderProps) => {
  const [sortBy, setSortBy] = useState("name");
  const [sortOrder, setSortOrder] = useState("asc");
  const [filterType, setFilterType] = useState("all");
  const [expiresAt, setExpiresAt] = useState(false);

  const handleSort = (field: string) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(field);
      setSortOrder("asc");
    }
  };
  const handleFilter = (type: string) => {
    setFilterType(type);
  };

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">
          {isLoading ? <Skeleton className="h-10 w-[250px]" /> : bucket!.name}
        </h1>
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
          <CustomDialog
            title="Share a file"
            description="Upload a file and share it safely"
            trigger={
              <Button>
                <PlusCircle className="mr-2 h-4 w-4" />
                Share a file
              </Button>
            }
            submitName="Share"
          >
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="file" className="">
                File
              </Label>
              <Input id="file" type="file" className="col-span-3" />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="username" className="">
                Password
              </Label>
              <Input
                id="username"
                defaultValue="0UymxETG$wc)7k8"
                className="col-span-3"
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="max-downloads" className="">
                Max downloads
              </Label>
              <Select>
                <SelectTrigger id="max-downloads" className="col-span-3">
                  <SelectValue
                    placeholder="Unlimited"
                    defaultValue="unlimited"
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unlimited">Unlimited</SelectItem>
                  <SelectItem value="1">1</SelectItem>
                  <SelectItem value="3">3</SelectItem>
                  <SelectItem value="5">5</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="expires-at" className="">
                Expires at
              </Label>
              <Switch
                id="expires-at"
                checked={expiresAt}
                onCheckedChange={setExpiresAt}
              />
            </div>
            {expiresAt && (
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="expires-at-date" className="">
                  Date
                </Label>
                <DatePickerDemo />
              </div>
            )}
          </CustomDialog>
        </div>
      </div>
    </div>
  );
};
