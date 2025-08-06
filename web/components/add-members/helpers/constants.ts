export const bucketGroups = [
  { id: "viewer", name: "Viewer", description: "Can view and download files" },
  {
    id: "contributor",
    name: "Contributor",
    description: "Can view, download and upload files",
  },
  {
    id: "owner",
    name: "Owner",
    description: "Can manage files and update the bucket",
  },
];

export const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;