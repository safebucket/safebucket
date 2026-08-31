import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChevronLeft, ChevronRight, LoaderCircle } from "lucide-react";
import { Document, Page, pdfjs } from "react-pdf";

import "react-pdf/dist/Page/TextLayer.css";

import { Button } from "@/components/ui/button";

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
  const [pageNumber, setPageNumber] = useState(1);

  useEffect(() => {
    setNumPages(undefined);
    setPageNumber(1);
  }, [url]);

  return (
    <div className="flex h-[70vh] w-full flex-col">
      <div className="flex min-h-0 flex-1 items-center justify-center overflow-auto p-4">
        <Document
          file={url}
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
        >
          <Page
            pageNumber={pageNumber}
            renderAnnotationLayer={false}
            renderTextLayer
            className="max-w-full shadow-sm"
          />
        </Document>
      </div>
      {numPages && (
        <div className="flex items-center justify-center gap-3 border-t px-4 py-2">
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => setPageNumber((currentPage) => currentPage - 1)}
            disabled={pageNumber === 1}
            aria-label={t("file_actions.pdf_previous_page")}
          >
            <ChevronLeft />
          </Button>
          <span className="text-sm text-muted-foreground">
            {t("file_actions.pdf_page", {
              current: pageNumber,
              total: numPages,
            })}
          </span>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => setPageNumber((currentPage) => currentPage + 1)}
            disabled={pageNumber === numPages}
            aria-label={t("file_actions.pdf_next_page")}
          >
            <ChevronRight />
          </Button>
        </div>
      )}
    </div>
  );
};
