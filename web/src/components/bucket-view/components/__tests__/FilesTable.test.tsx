// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { DndContext } from "@dnd-kit/core";
import { FilesTable } from "../FilesTable";
import type { IFile } from "@/types/file.ts";
import { FileStatus } from "@/types/file.ts";

const openItem = vi.hoisted(() => vi.fn());

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@tanstack/react-router", () => ({
  useParams: () => ({ bucketId: "bucket-1", folderId: undefined }),
}));

vi.mock("@/components/bucket-view/hooks/useOpenBucketItem", () => ({
  useOpenBucketItem: () => openItem,
}));

vi.mock("@/hooks/useConfig", () => ({
  useConfig: () => ({ trashRetentionDays: 30 }),
}));

vi.mock("@/hooks/usePermissions", () => ({
  useBucketPermissions: () => ({ isContributor: false, isOwner: false }),
}));

vi.mock("@/components/file-actions/hooks/useFileActions", () => ({
  useFileActions: () => ({ downloadFile: vi.fn(), deleteFile: vi.fn() }),
}));

vi.stubGlobal(
  "ResizeObserver",
  class {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
);

const FILE: IFile = {
  id: "file-1",
  name: "report.pdf",
  size: 1024,
  extension: "pdf",
  status: FileStatus.uploaded,
  created_at: "2026-01-01T00:00:00Z",
  deleted_at: null,
  expires_at: null,
};

const renderComponent = () =>
  render(
    <DndContext>
      <FilesTable
        items={[FILE]}
        selected={null}
        setSelected={vi.fn()}
        rowSelection={{}}
        setRowSelection={vi.fn()}
        enabled
        selectedIds={[]}
      />
    </DndContext>,
  );

const getRow = () => screen.getByText(FILE.name).closest("tr") as HTMLElement;

afterEach(() => {
  cleanup();
  openItem.mockClear();
});

describe("FilesTable", () => {
  it("opens the item on double click", () => {
    renderComponent();

    fireEvent.doubleClick(getRow());

    expect(openItem).toHaveBeenCalledWith(FILE);
  });

  it("does not let rows be text-selected, so double click is not swallowed", () => {
    renderComponent();

    expect(getRow().classList.contains("select-none")).toBe(true);
  });

  it("opens the file actions menu on right click", async () => {
    renderComponent();

    fireEvent.contextMenu(getRow());

    expect(await screen.findByText("file_actions.preview")).toBeTruthy();
  });
});
