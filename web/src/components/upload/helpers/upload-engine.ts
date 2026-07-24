import type { IUploadPresign } from "@/components/upload/helpers/types";

function sendWithXhr(
  xhr: XMLHttpRequest,
  body: XMLHttpRequestBodyInit,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> {
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

    xhr.send(body);
  });
}

function postFormWithXhr(
  url: string,
  body: Record<string, string>,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  const xhr = new XMLHttpRequest();
  xhr.open("POST", url, true);

  const formData = new FormData();
  Object.entries(body).forEach(([key, value]) => {
    formData.append(key, value);
  });
  formData.append("file", file);

  return sendWithXhr(xhr, formData, onProgress, signal);
}

function putWithXhr(
  url: string,
  headers: Record<string, string>,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  const xhr = new XMLHttpRequest();
  xhr.open("PUT", url, true);

  Object.entries(headers).forEach(([key, value]) => {
    xhr.setRequestHeader(key, value);
  });

  return sendWithXhr(xhr, file, onProgress, signal);
}

export const uploadToStorage = async (
  presigned: IUploadPresign,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> => {
  switch (presigned.method) {
    case "post":
      return postFormWithXhr(
        presigned.url,
        presigned.body,
        file,
        onProgress,
        signal,
      );
    case "put":
      return putWithXhr(
        presigned.url,
        presigned.headers,
        file,
        onProgress,
        signal,
      );
    default:
      throw new Error("Unsupported upload method");
  }
};
