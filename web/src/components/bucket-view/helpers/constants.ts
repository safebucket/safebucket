// Get tomorrow's date as minimum for expiration
const getTomorrowDate = () => {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  return tomorrow.toISOString().split("T")[0];
};

export const getShareFileFields = (t: (key: string) => string) => [
  { id: "files", label: t("bucket.header.file_label"), type: "file" as const, required: true },
  {
    id: "sharingOptionsSection",
    label: t("upload.settings.section_title"),
    type: "section" as const,
  },
  {
    id: "expiresAt",
    label: t("upload.settings.expiration_label"),
    type: "date" as const,
    min: getTomorrowDate(),
  },
  {
    id: "maxDownloads",
    label: t("upload.settings.max_downloads_label"),
    type: "number" as const,
    placeholder: t("upload.settings.max_downloads_placeholder"),
    min: 1,
    max: 10000,
  },
];
