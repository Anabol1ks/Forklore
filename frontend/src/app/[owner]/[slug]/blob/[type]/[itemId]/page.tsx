"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import axios from "axios";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { useAuthStore } from "@/store/auth";
import { FileText, Files, ArrowLeft } from "lucide-react";

interface Repository {
  id?: string;
  repo_id?: string;
  owner_id?: string;
  name: string;
  slug: string;
}

interface DocumentVersion {
  id?: string;
  version_id?: string;
  version_number?: number;
  change_summary?: string;
}

interface FileVersion {
  id?: string;
  version_id?: string;
  version_number?: number;
  change_summary?: string;
  storage_key?: string;
  mime_type?: string;
  size_bytes?: number;
}

function getId(obj: Record<string, unknown> | null | undefined, keys: string[]): string {
  if (!obj) return "";
  for (const key of keys) {
    const value = obj[key];
    if (typeof value === "string" && value.length > 0) {
      return value;
    }
  }
  return "";
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error)) {
    return (error.response?.data as { message?: string } | undefined)?.message || fallback;
  }
  return fallback;
}

export default function BlobPage() {
  const params = useParams<{ owner: string; slug: string; type: string; itemId: string }>();
  const router = useRouter();
  const { user } = useAuthStore();

  const owner = params.owner;
  const repoSlug = params.slug;
  const itemType = params.type;
  const itemId = params.itemId;

  const [repo, setRepo] = useState<Repository | null>(null);
  const [loading, setLoading] = useState(true);

  const [document, setDocument] = useState<Record<string, unknown> | null>(null);
  const [documentVersions, setDocumentVersions] = useState<DocumentVersion[]>([]);
  const [documentEditorContent, setDocumentEditorContent] = useState("");
  const [currentVersionContent, setCurrentVersionContent] = useState("");
  const [selectedDocumentVersionId, setSelectedDocumentVersionId] = useState<string | null>(null);
  const [selectedDocumentVersionContent, setSelectedDocumentVersionContent] = useState("");
  const [selectedDocumentVersionSummary, setSelectedDocumentVersionSummary] = useState("");
  const [documentChangeSummary, setDocumentChangeSummary] = useState("");
  const [documentViewMode, setDocumentViewMode] = useState<"preview" | "edit">("preview");

  const [file, setFile] = useState<Record<string, unknown> | null>(null);
  const [fileVersions, setFileVersions] = useState<FileVersion[]>([]);
  const [selectedFileVersionId, setSelectedFileVersionId] = useState<string | null>(null);
  const [selectedFileVersionDetails, setSelectedFileVersionDetails] = useState<Record<string, unknown> | null>(null);
  const [filePreviewText, setFilePreviewText] = useState("");
  const [filePreviewUrl, setFilePreviewUrl] = useState("");
  const [filePreviewLoading, setFilePreviewLoading] = useState(false);
  const [filePreviewError, setFilePreviewError] = useState("");
  const [resolvedPreviewMimeType, setResolvedPreviewMimeType] = useState("application/octet-stream");
  const [fileVersionStorageKey, setFileVersionStorageKey] = useState("");
  const [fileVersionMimeType, setFileVersionMimeType] = useState("application/octet-stream");
  const [fileVersionSize, setFileVersionSize] = useState("1024");
  const [fileVersionChangeSummary, setFileVersionChangeSummary] = useState("");

  const isOwner = !!user?.id && !!repo?.owner_id && user.id === repo.owner_id;
  const isDocument = itemType === "document";
  const isFile = itemType === "file";

  const hasDocumentChanges = documentEditorContent !== currentVersionContent;
  const documentDiffStats = useMemo(() => {
    const oldLines = currentVersionContent.split("\n");
    const newLines = documentEditorContent.split("\n");

    const oldSet = new Set(oldLines);
    const newSet = new Set(newLines);

    return {
      added: newLines.filter((line) => !oldSet.has(line)).length,
      removed: oldLines.filter((line) => !newSet.has(line)).length,
      oldCount: oldLines.length,
      newCount: newLines.length,
    };
  }, [currentVersionContent, documentEditorContent]);

  const activeFileVersion = useMemo(() => {
    if (selectedFileVersionDetails) {
      return selectedFileVersionDetails;
    }

    const currentVersionId = typeof file?.current_version_id === "string" ? file.current_version_id : "";
    const byCurrent = fileVersions.find((v) => (v.version_id || v.id) === currentVersionId);
    if (byCurrent) {
      return byCurrent as unknown as Record<string, unknown>;
    }

    if (fileVersions.length === 0) {
      return null;
    }

    const sorted = [...fileVersions].sort((a, b) => (b.version_number || 0) - (a.version_number || 0));
    return sorted[0] as unknown as Record<string, unknown>;
  }, [selectedFileVersionDetails, file, fileVersions]);

  const activeFileVersionId = useMemo(() => {
    const id = selectedFileVersionId || getId(activeFileVersion, ["version_id", "id"]);
    return id || "";
  }, [selectedFileVersionId, activeFileVersion]);

  const activeFileStorageKey = typeof activeFileVersion?.storage_key === "string" ? activeFileVersion.storage_key : "";
  const activeFileMime = typeof activeFileVersion?.mime_type === "string" ? activeFileVersion.mime_type : "application/octet-stream";
  const effectivePreviewMimeType = resolvedPreviewMimeType || activeFileMime;
  const isPdfPreview = effectivePreviewMimeType.includes("pdf") || /\.pdf($|\?)/i.test(activeFileStorageKey);
  const isTextPreview = effectivePreviewMimeType.startsWith("text/") || effectivePreviewMimeType.includes("json") || effectivePreviewMimeType.includes("xml") || /\.(txt|md|log|csv|json|xml|yaml|yml)($|\?)/i.test(activeFileStorageKey);

  const fetchRepo = useCallback(async (): Promise<Repository> => {
    const bySlug = await api.get(`/users/${owner}/repositories/${repoSlug}`);
    const slugRepo = (bySlug.data.repository || bySlug.data) as Repository;
    const repoId = getId(slugRepo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) {
      throw new Error("Repository id is missing");
    }

    const byID = await api.get(`/repositories/${repoId}`);
    const repoData = (byID.data.repository || slugRepo) as Repository;
    setRepo(repoData);
    return repoData;
  }, [owner, repoSlug]);

  const fetchDocument = useCallback(async () => {
    const [detailRes, versionsRes] = await Promise.all([
      api.get(`/documents/${itemId}`),
      api.get(`/documents/${itemId}/versions`),
    ]);

    const detailPayload = detailRes.data as { document?: Record<string, unknown> };
    const documentState = (detailPayload.document || detailRes.data) as Record<string, unknown>;
    const draft = documentState.draft as { content?: string } | undefined;
    const currentVersion = documentState.current_version as { content?: string } | undefined;

    const baseline = currentVersion?.content || "";
    const editable = draft?.content || baseline;

    setDocument(documentState);
    setDocumentVersions((versionsRes.data.versions || []) as DocumentVersion[]);
    setCurrentVersionContent(baseline);
    setDocumentEditorContent(editable);
    setSelectedDocumentVersionId(null);
    setSelectedDocumentVersionContent("");
    setSelectedDocumentVersionSummary("");
    setDocumentViewMode("preview");
  }, [itemId]);

  const fetchFile = useCallback(async () => {
    const [detailRes, versionsRes] = await Promise.all([
      api.get(`/files/${itemId}`),
      api.get(`/files/${itemId}/versions`),
    ]);

    const fileState = (detailRes.data.file || detailRes.data) as Record<string, unknown>;
    setFile(fileState);
    setFileVersions((versionsRes.data.versions || []) as FileVersion[]);
    setSelectedFileVersionId(null);
    setSelectedFileVersionDetails(null);
  }, [itemId]);

  const handleViewDocumentVersion = async (versionId: string) => {
    try {
      const versionRes = await api.get(`/document-versions/${versionId}`);
      const versionPayload = versionRes.data as {
        version?: { content?: string; change_summary?: string };
      };
      const version = versionPayload.version || {};
      setSelectedDocumentVersionId(versionId);
      setSelectedDocumentVersionContent(version.content || "");
      setSelectedDocumentVersionSummary(version.change_summary || "");
      setDocumentViewMode("preview");
    } catch (error) {
      toast.error(getErrorMessage(error, "Не удалось загрузить версию документа"));
    }
  };

  const handleUseCurrentDocumentVersion = () => {
    setSelectedDocumentVersionId(null);
    setSelectedDocumentVersionContent("");
    setSelectedDocumentVersionSummary("");
  };

  const handleViewFileVersion = async (versionId: string) => {
    try {
      const versionRes = await api.get(`/file-versions/${versionId}`);
      const versionPayload = versionRes.data as { version?: Record<string, unknown> };
      setSelectedFileVersionId(versionId);
      setSelectedFileVersionDetails((versionPayload.version || null) as Record<string, unknown> | null);
    } catch (error) {
      toast.error(getErrorMessage(error, "Не удалось загрузить версию файла"));
    }
  };

  const handleUseCurrentFileVersion = () => {
    setSelectedFileVersionId(null);
    setSelectedFileVersionDetails(null);
  };

  useEffect(() => {
    const run = async () => {
      try {
        setLoading(true);
        await fetchRepo();

        if (isDocument) {
          await fetchDocument();
        } else if (isFile) {
          await fetchFile();
        } else {
          throw new Error("Unsupported blob type");
        }
      } catch (error) {
        toast.error(getErrorMessage(error, "Не удалось загрузить содержимое"));
      } finally {
        setLoading(false);
      }
    };

    void run();
  }, [fetchRepo, fetchDocument, fetchFile, isDocument, isFile]);

  useEffect(() => {
    let objectUrl: string | null = null;

    const run = async () => {
      if (!isFile || !file) {
        setFilePreviewText("");
        setFilePreviewUrl("");
        setFilePreviewError("");
        setFilePreviewLoading(false);
        return;
      }

      try {
        setFilePreviewLoading(true);
        setFilePreviewError("");
        setFilePreviewText("");

        const params = activeFileVersionId ? { version_id: activeFileVersionId } : undefined;
        const response = await api.get(`/files/${itemId}/content`, {
          params,
          responseType: "blob",
        });

        const blob = response.data as Blob;
        const headerMime = (response.headers["content-type"] as string | undefined) || "";
        const detectedMime = headerMime || blob.type || activeFileMime || "application/octet-stream";
        setResolvedPreviewMimeType(detectedMime);

        if (detectedMime.startsWith("text/") || detectedMime.includes("json") || detectedMime.includes("xml")) {
          const text = await blob.text();
          setFilePreviewText(text);
          setFilePreviewUrl("");
        } else {
          objectUrl = URL.createObjectURL(blob);
          setFilePreviewUrl(objectUrl);
          setFilePreviewText("");
        }
      } catch (error) {
        setFilePreviewText("");
        setFilePreviewUrl("");
        setResolvedPreviewMimeType(activeFileMime);
        setFilePreviewError(getErrorMessage(error, "Не удалось загрузить содержимое файла"));
      } finally {
        setFilePreviewLoading(false);
      }
    };

    void run();

    return () => {
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [isFile, file, itemId, activeFileVersionId, activeFileMime]);

  const handleSaveDraft = async () => {
    if (!isOwner || !isDocument) return;

    try {
      await api.patch(`/documents/${itemId}/draft`, { content: documentEditorContent });
      toast.success("Черновик сохранен");
      await fetchDocument();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка сохранения черновика"));
    }
  };

  const handleCreateVersion = async () => {
    if (!isOwner || !isDocument) return;
    if (!documentEditorContent.trim()) {
      toast.error("Нельзя создать версию с пустым содержимым");
      return;
    }
    if (!hasDocumentChanges) {
      toast.error("Нет изменений относительно текущей версии");
      return;
    }

    try {
      await api.post(`/documents/${itemId}/versions`, {
        content: documentEditorContent,
        change_summary: documentChangeSummary,
      });
      toast.success("Версия документа создана");
      setDocumentChangeSummary("");
      await fetchDocument();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка создания версии"));
    }
  };

  const handleRestoreDocumentVersion = async (versionId: string) => {
    if (!isOwner || !isDocument) return;

    try {
      await api.post(`/documents/${itemId}/versions/${versionId}/restore`, {});
      toast.success("Версия документа восстановлена");
      await fetchDocument();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка восстановления версии"));
    }
  };

  const handleAddFileVersion = async () => {
    if (!isOwner || !isFile) return;

    const size = Number.parseInt(fileVersionSize, 10);
    if (!Number.isFinite(size) || size <= 0) {
      toast.error("Размер версии файла должен быть положительным числом");
      return;
    }

    const storageKey = fileVersionStorageKey.trim() || `files/version_${Date.now()}`;

    try {
      await api.post(`/files/${itemId}/versions`, {
        storage_key: storageKey,
        mime_type: fileVersionMimeType || "application/octet-stream",
        size_bytes: size,
        change_summary: fileVersionChangeSummary,
      });
      toast.success("Версия файла добавлена");
      setFileVersionStorageKey("");
      setFileVersionChangeSummary("");
      await fetchFile();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка добавления версии файла"));
    }
  };

  const handleRestoreFileVersion = async (versionId: string) => {
    if (!isOwner || !isFile) return;

    try {
      await api.post(`/files/${itemId}/versions/${versionId}/restore`, {});
      toast.success("Версия файла восстановлена");
      await fetchFile();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка восстановления версии файла"));
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-80" />
        <Skeleton className="h-80 w-full" />
      </div>
    );
  }

  if (!repo) {
    return <div className="text-center py-20 text-xl font-bold">Репозиторий не найден</div>;
  }

  const activeDocumentPreview = selectedDocumentVersionId ? selectedDocumentVersionContent : documentEditorContent;
  const activeDocumentVersionLabel = selectedDocumentVersionId ? "Просматривается историческая версия" : "Текущая версия/черновик";

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between border-b pb-4">
        <div className="flex items-center gap-2 text-xl font-semibold">
          <Button variant="ghost" size="sm" onClick={() => router.push(`/${owner}/${repoSlug}`)}>
            <ArrowLeft className="h-4 w-4 mr-1" /> Назад
          </Button>
          <span className="text-primary">{owner}</span>
          <span className="text-muted-foreground">/</span>
          <span>{repo.name}</span>
        </div>
      </div>

      {isDocument && document ? (
        <div className="space-y-4">
          <div className="border rounded-md p-4 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold flex items-center gap-2"><FileText className="h-5 w-5" /> {String(document.title || "Документ")}</h2>
              <span className="text-sm text-muted-foreground">Версий: {documentVersions.length}</span>
            </div>

            <div className="flex items-center gap-2 border-b pb-2">
              <Button size="sm" variant={documentViewMode === "preview" ? "default" : "outline"} onClick={() => setDocumentViewMode("preview")}>Preview</Button>
              {isOwner ? <Button size="sm" variant={documentViewMode === "edit" ? "default" : "outline"} onClick={() => setDocumentViewMode("edit")}>Edit</Button> : null}
            </div>

            <div className="text-sm text-muted-foreground">
              {activeDocumentVersionLabel}
              {selectedDocumentVersionId ? (
                <>
                  {": "}
                  <span className="font-medium text-foreground">{selectedDocumentVersionId}</span>
                  <Button variant="ghost" size="sm" className="ml-2" onClick={handleUseCurrentDocumentVersion}>
                    Вернуться к текущей
                  </Button>
                </>
              ) : null}
            </div>

            {documentViewMode === "preview" ? (
              <div className="rounded-md border p-4 min-h-40 prose prose-invert max-w-none">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{activeDocumentPreview || "_Пустой документ_"}</ReactMarkdown>
              </div>
            ) : (
              <>
                <Textarea value={documentEditorContent} onChange={(e) => setDocumentEditorContent(e.target.value)} rows={16} placeholder="Содержимое документа" />
                <Input value={documentChangeSummary} onChange={(e) => setDocumentChangeSummary(e.target.value)} placeholder="Описание изменений" />
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={handleSaveDraft}>Сохранить черновик</Button>
                  <Button size="sm" onClick={handleCreateVersion}>Создать версию</Button>
                </div>
              </>
            )}

            {selectedDocumentVersionId ? (
              <div className="border rounded-md p-3 text-sm">
                <div className="font-medium mb-1">Выбранная версия: {selectedDocumentVersionId}</div>
                <div className="text-muted-foreground">{selectedDocumentVersionSummary || "Без описания"}</div>
              </div>
            ) : null}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
            <div className="border rounded-md p-3">
              <div className="font-medium mb-2">Сравнение с текущей версией</div>
              <div className="text-muted-foreground">Строк в текущей: {documentDiffStats.oldCount}</div>
              <div className="text-muted-foreground">Строк в черновике: {documentDiffStats.newCount}</div>
              <div className="text-green-600">Добавлено строк: {documentDiffStats.added}</div>
              <div className="text-red-600">Удалено строк: {documentDiffStats.removed}</div>
              <div className="mt-2 font-medium">{hasDocumentChanges ? "Есть изменения" : "Изменений нет"}</div>
            </div>
            <div className="border rounded-md p-3">
              <div className="font-medium mb-2">Текущая версия (reference)</div>
              <pre className="text-xs overflow-auto max-h-40 whitespace-pre-wrap">{currentVersionContent || "Пусто"}</pre>
            </div>
          </div>

          <div className="border rounded-md p-4 space-y-2">
            <h3 className="font-medium">История версий</h3>
            {documentVersions.length > 0 ? documentVersions.map((version) => {
              const versionId = version.version_id || version.id;
              if (!versionId) return null;
              return (
                <div key={versionId} className="flex items-center justify-between border rounded-md p-2 text-sm">
                  <div>
                    <div className="font-medium">Версия #{version.version_number || "-"}</div>
                    <div className="text-muted-foreground">{version.change_summary || "Без описания"}</div>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={() => void handleViewDocumentVersion(versionId)}>Просмотр</Button>
                    {isOwner ? <Button variant="outline" size="sm" onClick={() => void handleRestoreDocumentVersion(versionId)}>Восстановить</Button> : null}
                  </div>
                </div>
              );
            }) : <div className="text-sm text-muted-foreground">Версий пока нет</div>}
          </div>
        </div>
      ) : null}

      {isFile && file ? (
        <div className="space-y-4">
          <div className="border rounded-md p-4 space-y-3">
            <h2 className="text-lg font-semibold flex items-center gap-2"><Files className="h-5 w-5" /> {String(file.file_name || "Файл")}</h2>
            <div className="text-sm text-muted-foreground">Тип: {effectivePreviewMimeType}</div>
            <div className="text-sm text-muted-foreground">Источник: {activeFileStorageKey || "не задан"}</div>
            {activeFileVersionId ? <div className="text-sm text-muted-foreground">Версия для просмотра: {activeFileVersionId}</div> : null}

            {filePreviewLoading ? <div className="text-sm text-muted-foreground">Загрузка содержимого...</div> : null}
            {filePreviewError ? <div className="text-sm text-red-600">{filePreviewError}</div> : null}

            {!filePreviewLoading && !filePreviewError && isPdfPreview && filePreviewUrl ? (
              <div className="space-y-2">
                <div className="text-sm font-medium">PDF preview</div>
                <iframe
                  src={filePreviewUrl}
                  title="pdf-preview"
                  className="w-full h-[70vh] border rounded-md"
                />
              </div>
            ) : null}

            {!filePreviewLoading && !filePreviewError && isTextPreview ? (
              <div className="space-y-2">
                <div className="text-sm font-medium">Text preview</div>
                <pre className="text-xs overflow-auto max-h-[70vh] whitespace-pre-wrap border rounded-md p-3 bg-muted/30">{filePreviewText || "Пустой файл"}</pre>
              </div>
            ) : null}

            {!filePreviewLoading && !filePreviewError && !isPdfPreview && !isTextPreview && filePreviewUrl ? (
              <div className="text-sm text-muted-foreground">Для этого MIME пока доступен только переход по ссылке на контент.</div>
            ) : null}

            {filePreviewUrl ? (
              <div>
                <a href={filePreviewUrl} target="_blank" rel="noreferrer" className="text-sm underline text-primary">
                  Открыть файл в новой вкладке
                </a>
              </div>
            ) : null}

            {selectedFileVersionId && selectedFileVersionDetails ? (
              <div className="border rounded-md p-3 text-sm space-y-1">
                <div className="font-medium">Просмотр версии файла: {selectedFileVersionId}</div>
                <div className="text-muted-foreground">Storage key: {String(selectedFileVersionDetails.storage_key || "-")}</div>
                <div className="text-muted-foreground">MIME: {String(selectedFileVersionDetails.mime_type || "-")}</div>
                <div className="text-muted-foreground">Размер: {String(selectedFileVersionDetails.size_bytes || "-")} bytes</div>
                <div className="text-muted-foreground">Описание: {String(selectedFileVersionDetails.change_summary || "Без описания")}</div>
                <Button variant="ghost" size="sm" onClick={handleUseCurrentFileVersion}>Вернуться к текущей</Button>
              </div>
            ) : null}
          </div>

          {isOwner ? (
            <div className="border rounded-md p-4 space-y-2">
              <h3 className="font-medium">Добавить версию файла</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                <Input value={fileVersionStorageKey} onChange={(e) => setFileVersionStorageKey(e.target.value)} placeholder="Storage key" />
                <Input value={fileVersionMimeType} onChange={(e) => setFileVersionMimeType(e.target.value)} placeholder="MIME type" />
                <Input value={fileVersionSize} onChange={(e) => setFileVersionSize(e.target.value)} placeholder="Размер в байтах" />
                <Input value={fileVersionChangeSummary} onChange={(e) => setFileVersionChangeSummary(e.target.value)} placeholder="Описание изменений" />
              </div>
              <Button size="sm" onClick={handleAddFileVersion}>Добавить версию</Button>
            </div>
          ) : null}

          <div className="border rounded-md p-4 space-y-2">
            <h3 className="font-medium">История версий файла</h3>
            {fileVersions.length > 0 ? fileVersions.map((version) => {
              const versionId = version.version_id || version.id;
              if (!versionId) return null;
              return (
                <div key={versionId} className="flex items-center justify-between border rounded-md p-2 text-sm">
                  <div>
                    <div className="font-medium">Версия #{version.version_number || "-"}</div>
                    <div className="text-muted-foreground">{version.change_summary || "Без описания"}</div>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={() => void handleViewFileVersion(versionId)}>Просмотр</Button>
                    {isOwner ? <Button variant="outline" size="sm" onClick={() => void handleRestoreFileVersion(versionId)}>Восстановить</Button> : null}
                  </div>
                </div>
              );
            }) : <div className="text-sm text-muted-foreground">Версий пока нет</div>}
          </div>
        </div>
      ) : null}
    </div>
  );
}
