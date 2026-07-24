import type { IUploadPresign } from "@/components/upload/helpers/types";

function postFormWithXhr(
  url: string,
  body: Record<string, string>,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  const xhr = new XMLHttpRequest();

  return new Promise((resolve, reject) => {
    if (signal) {
      signal.addEventListener("abort", () => {
        xhr.abort();
        reject(new Error("Upload cancelled"));
      });
    }

    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) {
        onProgress(Math.round((event.loaded / event.total) * 100));
      }
    });

    xhr.addEventListener("loadend", () => {
      if (xhr.readyState === 4) {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(`Upload failed with status ${xhr.status}`));
        }
      }
    });

    xhr.addEventListener("error", () => {
      reject(new Error("Network error during upload"));
    });

    xhr.addEventListener("abort", () => {
      reject(new Error("Upload cancelled"));
    });

    xhr.open("POST", url, true);
    const formData = new FormData();
    Object.entries(body).forEach(([key, value]) => {
      formData.append(key, value);
    });
    formData.append("file", file);

    xhr.send(formData);
  });
}

export const uploadToStorage = async (
  presigned: IUploadPresign,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> => {
  if (presigned.method === "post") {
    return postFormWithXhr(
      presigned.url,
      presigned.body,
      file,
      onProgress,
      signal,
    );
  }

  throw new Error("Unsupported upload method");
};
