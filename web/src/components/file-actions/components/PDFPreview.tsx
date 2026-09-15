import { useState } from "react";
import { useTranslation } from "react-i18next";
import { LoaderCircle } from "lucide-react";
import { Document, Page, pdfjs } from "react-pdf";

import "react-pdf/dist/Page/TextLayer.css";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url,
).toString();

const pdfOptions = {
  enableXfa: false,
  isEvalSupported: false,
};

interface IPDFPreviewProps {
  url: string;
}

export const PDFPreview = ({ url }: IPDFPreviewProps) => {
  const { t } = useTranslation();
  const [numPages, setNumPages] = useState<number>();

  return (
    <div className="flex h-[70vh] w-full flex-col">
      <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-4">
        <Document
          file={url}
          suspense={false}
          loading={
            <LoaderCircle className="h-8 w-8 animate-spin text-muted-foreground" />
          }
          error={
            <p className="p-6 text-sm text-destructive">
              {t("file_actions.preview_failed")}
            </p>
          }
          onLoadSuccess={({ numPages: loadedNumPages }) =>
            setNumPages(loadedNumPages)
          }
          options={pdfOptions}
          className="flex flex-col items-center gap-4"
        >
          {Array.from({ length: numPages ?? 0 }, (_, index) => (
            <Page
              key={index}
              pageNumber={index + 1}
              renderAnnotationLayer={false}
              renderTextLayer
              className="max-w-full shadow-sm"
            />
          ))}
        </Document>
      </div>
    </div>
  );
};
