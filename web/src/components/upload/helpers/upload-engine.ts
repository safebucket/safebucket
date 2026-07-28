import type {
  IFilePartURL,
  IUploadPresign,
} from "@/components/upload/helpers/types";

const MAX_CONCURRENT_PARTS = 4;
const MAX_PART_ATTEMPTS = 3;
const RETRY_DELAYS_MS = [1000, 2000, 4000];

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
  body: Array<Record<string, string>>,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  const xhr = new XMLHttpRequest();
  xhr.open("POST", url, true);

  const formData = new FormData();
  body.forEach((fields) => {
    Object.entries(fields).forEach(([key, value]) => {
      formData.append(key, value);
    });
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

function putPartWithXhr(
  url: string,
  headers: Record<string, string>,
  body: Blob,
  onLoaded: (loaded: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  const xhr = new XMLHttpRequest();
  xhr.open("PUT", url, true);

  Object.entries(headers).forEach(([key, value]) => {
    xhr.setRequestHeader(key, value);
  });

  return new Promise((resolve, reject) => {
    const onAbort = () => {
      xhr.abort();
      reject(new Error("Upload cancelled"));
    };

    if (signal) {
      if (signal.aborted) {
        onAbort();
        return;
      }
      signal.addEventListener("abort", onAbort);
    }

    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) {
        onLoaded(event.loaded);
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

async function multipartUpload(
  parts: Array<IFilePartURL>,
  file: File,
  onProgress: (progress: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  let offset = 0;
  const tasks = [...parts]
    .sort((a, b) => a.part_number - b.part_number)
    .map((part) => {
      const start = offset;
      offset += part.size;
      return { url: part.url, headers: part.headers ?? {}, start, end: offset };
    });

  const loaded = new Array<number>(tasks.length).fill(0);

  const reportProgress = () => {
    const total = loaded.reduce((sum, n) => sum + n, 0);
    onProgress(Math.min(100, Math.round((total / file.size) * 100)));
  };

  const uploadTask = async (index: number): Promise<void> => {
    const { url, headers, start, end } = tasks[index];
    const blob = file.slice(start, end);

    for (let attempt = 1; attempt <= MAX_PART_ATTEMPTS; attempt++) {
      if (signal?.aborted) {
        throw new Error("Upload cancelled");
      }

      try {
        await putPartWithXhr(
          url,
          headers,
          blob,
          (n) => {
            loaded[index] = n;
            reportProgress();
          },
          signal,
        );
        return;
      } catch (err) {
        if (err instanceof Error && err.message === "Upload cancelled") {
          throw err;
        }
        if (attempt >= MAX_PART_ATTEMPTS) {
          throw err;
        }
        loaded[index] = 0;
        reportProgress();
        await new Promise((resolve) =>
          setTimeout(resolve, RETRY_DELAYS_MS[attempt - 1] ?? 4000),
        );
      }
    }
  };

  let cursor = 0;
  const nextIndex = (): number | null => {
    if (cursor >= tasks.length) return null;
    const index = cursor;
    cursor += 1;
    return index;
  };

  const worker = async (): Promise<void> => {
    for (;;) {
      if (signal?.aborted) {
        throw new Error("Upload cancelled");
      }
      const index = nextIndex();
      if (index === null) return;
      await uploadTask(index);
    }
  };

  const workerCount = Math.min(MAX_CONCURRENT_PARTS, tasks.length);
  await Promise.all(Array.from({ length: workerCount }, () => worker()));
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
      if (presigned.parts.length > 1) {
        return multipartUpload(presigned.parts, file, onProgress, signal);
      }
      return putWithXhr(
        presigned.parts[0].url,
        presigned.parts[0].headers ?? {},
        file,
        onProgress,
        signal,
      );
    default:
      throw new Error("Unsupported upload method");
  }
};
