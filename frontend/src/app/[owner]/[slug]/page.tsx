"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Book, FileText, GitFork, Settings, Star, Files } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { useAuthStore } from "@/store/auth";
import axios from "axios";
import Link from "next/link";

interface Repository {
  id?: string;
  repo_id?: string;
  name: string;
  description?: string;
  visibility: string;
  owner_id?: string;
  owner_username?: string;
  slug: string;
  parent_repo_id?: string;
  created_at?: string;
}

type ForkVisibility = "public" | "private";

interface DocumentItem {
  id?: string;
  document_id?: string;
  title?: string;
  slug?: string;
  status?: string;
  updated_at?: string;
}

interface FileItem {
  id?: string;
  file_id?: string;
  file_name?: string;
  mime_type?: string;
  size?: number;
  size_bytes?: number;
}

interface VersionItem {
  id?: string;
  version_id?: string;
  version_number?: number;
  content?: string;
  change_summary?: string;
  created_at?: string;
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

function normalizeForkSlug(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function SectionState({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="p-6 text-center text-muted-foreground border rounded-md">
      <p className="font-medium">{title}</p>
      <p className="text-sm mt-1">{description}</p>
    </div>
  );
}

function Modal({
  open,
  title,
  onClose,
  children,
  footer,
}: {
  open: boolean;
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="max-h-[90vh] w-full max-w-2xl overflow-hidden rounded-lg border bg-background shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b p-4">
          <h3 className="text-base font-semibold sm:text-lg">{title}</h3>
          <Button variant="ghost" size="sm" onClick={onClose}>Закрыть</Button>
        </div>
        <div className="overflow-y-auto p-4 space-y-4">{children}</div>
        {footer ? <div className="border-t p-4 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">{footer}</div> : null}
      </div>
    </div>
  );
}

export default function RepositoryPage() {
  const params = useParams();
  const router = useRouter();
  const owner = params.owner as string;
  const slug = params.slug as string;
  const { user } = useAuthStore();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [parentRepo, setParentRepo] = useState<Repository | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorState, setErrorState] = useState<"not-found" | "forbidden" | null>(null);

  const [documents, setDocuments] = useState<DocumentItem[]>([]);
  const [files, setFiles] = useState<FileItem[]>([]);
  const [forks, setForks] = useState<Repository[]>([]);

  const [selectedDocument, setSelectedDocument] = useState<Record<string, unknown> | null>(null);
  const [selectedFile, setSelectedFile] = useState<Record<string, unknown> | null>(null);
  const [documentVersions, setDocumentVersions] = useState<VersionItem[]>([]);
  const [fileVersions, setFileVersions] = useState<VersionItem[]>([]);
  const [newDocTitle, setNewDocTitle] = useState("");
  const [newDocContent, setNewDocContent] = useState("");
  const [newFileName, setNewFileName] = useState("");
  const [selectedUploadFile, setSelectedUploadFile] = useState<File | null>(null);
  const [documentEditorContent, setDocumentEditorContent] = useState("");
  const [currentVersionContent, setCurrentVersionContent] = useState("");
  const [documentViewMode, setDocumentViewMode] = useState<"preview" | "edit">("preview");
  const [documentChangeSummary, setDocumentChangeSummary] = useState("");
  const [fileVersionStorageKey, setFileVersionStorageKey] = useState("");
  const [fileVersionMimeType, setFileVersionMimeType] = useState("application/octet-stream");
  const [fileVersionSize, setFileVersionSize] = useState("1024");
  const [fileVersionChangeSummary, setFileVersionChangeSummary] = useState("");
  const [editRepoName, setEditRepoName] = useState("");
  const [editRepoDescription, setEditRepoDescription] = useState("");
  const [documentPanelError, setDocumentPanelError] = useState("");
  const [filePanelError, setFilePanelError] = useState("");
  const [isCreateDocumentModalOpen, setCreateDocumentModalOpen] = useState(false);
  const [isCreateFileModalOpen, setCreateFileModalOpen] = useState(false);
  const [isEditRepoModalOpen, setEditRepoModalOpen] = useState(false);
  const [isForkModalOpen, setForkModalOpen] = useState(false);
  const [forkName, setForkName] = useState("");
  const [forkDescription, setForkDescription] = useState("");
  const [forkVisibility, setForkVisibility] = useState<ForkVisibility | "">("");
  const [isForking, setForking] = useState(false);
  const [starsCount, setStarsCount] = useState(0);
  const [isStarred, setIsStarred] = useState(false);
  const [isStarLoading, setStarLoading] = useState(false);
  const isOwner = !!user?.id && !!repo?.owner_id && user.id === repo.owner_id;
  const hasDocumentChanges = documentEditorContent !== currentVersionContent;
  const effectiveForkSlug = normalizeForkSlug(forkName) || normalizeForkSlug(repo?.slug || "") || "repo";
  const forkOwner = user?.username || "you";

  const documentDiffStats = useMemo(() => {
    const oldLines = currentVersionContent.split("\n");
    const newLines = documentEditorContent.split("\n");

    const oldSet = new Set(oldLines);
    const newSet = new Set(newLines);

    const added = newLines.filter((line) => !oldSet.has(line)).length;
    const removed = oldLines.filter((line) => !newSet.has(line)).length;

    return {
      added,
      removed,
      oldCount: oldLines.length,
      newCount: newLines.length,
    };
  }, [currentVersionContent, documentEditorContent]);

  const fetchRepo = useCallback(async () => {
    try {
      setLoading(true);
      setErrorState(null);
      const response = await api.get(`/users/${owner}/repositories/${slug}`);
      const repoData = (response.data.repository || response.data) as Repository;
      const repoId = getId(repoData as unknown as Record<string, unknown>, ["id", "repo_id"]);

      if (!repoId) {
        throw new Error("Repository id is missing in response");
      }

      const [repoByIdResponse, docsRes, filesRes, forksRes] = await Promise.all([
        api.get(`/repositories/${repoId}`),
        api.get(`/repositories/${repoId}/documents`),
        api.get(`/repositories/${repoId}/files`),
        api.get(`/repositories/${repoId}/forks`),
      ]);

      const resolvedRepo = (repoByIdResponse.data.repository || repoData) as Repository;
      setRepo(resolvedRepo);

      const parentRepoId = getId(resolvedRepo as unknown as Record<string, unknown>, ["parent_repo_id"]);
      if (parentRepoId) {
        try {
          const parentRepoResponse = await api.get(`/repositories/${parentRepoId}`);
          setParentRepo((parentRepoResponse.data.repository || parentRepoResponse.data) as Repository);
        } catch {
          setParentRepo(null);
        }
      } else {
        setParentRepo(null);
      }

      setDocuments((docsRes.data.documents || []) as DocumentItem[]);
      setFiles((filesRes.data.files || []) as FileItem[]);
      setForks((forksRes.data.repositories || []) as Repository[]);

      try {
        const starRes = await api.get(`/repositories/${repoId}/star`);
        setStarsCount(Number(starRes.data.stars_count || 0));
        setIsStarred(Boolean(starRes.data.starred));
      } catch {
        setStarsCount(0);
        setIsStarred(false);
      }
    } catch (error) {
      console.error(error);
      if (axios.isAxiosError(error)) {
        const status = error.response?.status;
        if (status === 404) {
          setErrorState("not-found");
        } else if (status === 403) {
          setErrorState("forbidden");
        }
      }
      toast.error(getErrorMessage(error, "Ошибка при загрузке репозитория"));
    } finally {
      setLoading(false);
    }
  }, [owner, slug]);

  useEffect(() => {
    if (owner && slug) {
      void fetchRepo();
    }
  }, [owner, slug, fetchRepo]);

  useEffect(() => {
    if (!repo) return;
    setEditRepoName(repo.name || "");
    setEditRepoDescription(repo.description || "");
  }, [repo]);

  const openForkModal = () => {
    if (!repo) return;

    if (!user?.id) {
      toast.error("Для форка нужно войти в аккаунт");
      router.push("/login");
      return;
    }

    setForkName("");
    setForkDescription("");
    setForkVisibility("");
    setForkModalOpen(true);
  };

  const handleFork = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;
    if (!forkVisibility) {
      toast.error("Выберите видимость форка");
      return;
    }

    try {
      setForking(true);
      const response = await api.post(`/repositories/${repoId}/fork`, {
        name: forkName.trim(),
        slug: "",
        description: forkDescription.trim(),
        visibility: forkVisibility,
      });
      toast.success("Репозиторий успешно форкнут!");
      const newRepo = (response.data.repository || response.data) as Repository;
      setForkModalOpen(false);
      if (user?.username && newRepo.slug) {
        router.push(`/${user.username}/${newRepo.slug}`);
      }
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка при форке репозитория"));
    } finally {
      setForking(false);
    }
  };

  const handleToggleStar = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId || isStarLoading) return;

    if (!user?.id) {
      toast.error("Для Star нужно войти в аккаунт");
      router.push("/login");
      return;
    }

    try {
      setStarLoading(true);
      const res = await api.post(`/repositories/${repoId}/star`);
      setIsStarred(Boolean(res.data.starred));
      setStarsCount(Number(res.data.stars_count || 0));
    } catch (error) {
      toast.error(getErrorMessage(error, "Не удалось обновить star"));
    } finally {
      setStarLoading(false);
    }
  };

  const handleAddFile = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    if (!selectedUploadFile) {
      toast.error("Выберите файл для загрузки");
      return;
    }

    try {
      const formData = new FormData();
      formData.append("file", selectedUploadFile);

      await api.post(`/repositories/${repoId}/files/upload`, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });

      toast.success("Файл добавлен!");
      setNewFileName("");
      setSelectedUploadFile(null);
      setCreateFileModalOpen(false);
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка добавления файла"));
    }
  };

  const handleSelectUploadFile = (file: File | null) => {
    setSelectedUploadFile(file);
    if (!file) return;

    setNewFileName(file.name || "");
  };

  const handleEditRepo = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    const newName = editRepoName.trim();
    if (!newName) {
      toast.error("Название репозитория не может быть пустым");
      return;
    }
    const newDesc = editRepoDescription;

    try {
      await api.patch(`/repositories/${repoId}`, { name: newName, description: newDesc });
      toast.success("Репозиторий обновлен!");
      setEditRepoModalOpen(false);
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка обновления"));
    }
  };

  const handleDelete = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    if (!confirm("Вы уверены, что хотите удалить репозиторий? Это действие необратимо.")) return;

    try {
      await api.delete(`/repositories/${repoId}`);
      toast.success("Репозиторий удален");
      router.push("/");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка при удалении"));
    }
  };

  const handleCreateDocument = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    const title = newDocTitle.trim();
    if (!title) {
      toast.error("Введите название документа");
      return;
    }

    try {
      await api.post(`/repositories/${repoId}/documents`, {
        title,
        initial_content: newDocContent,
      });
      toast.success("Документ создан!");
      setNewDocTitle("");
      setNewDocContent("");
      setCreateDocumentModalOpen(false);
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка при создании документа"));
    }
  };

  const handleDocumentDetails = async (doc: DocumentItem) => {
    const documentId = getId(doc as unknown as Record<string, unknown>, ["id", "document_id"]);
    if (!documentId) return;

    try {
      const [detailRes, versionsRes] = await Promise.all([
        api.get(`/documents/${documentId}`),
        api.get(`/documents/${documentId}/versions`),
      ]);
      const detailPayload = detailRes.data as { document?: Record<string, unknown> };
      const documentState = (detailPayload.document || detailRes.data) as Record<string, unknown>;
      const draft = documentState.draft as { content?: string } | undefined;
      const currentVersion = documentState.current_version as { content?: string } | undefined;

      setSelectedDocument(documentState);
      setDocumentVersions((versionsRes.data.versions || []) as VersionItem[]);
      const baseline = currentVersion?.content || "";
      const editable = draft?.content || baseline;
      setCurrentVersionContent(baseline);
      setDocumentEditorContent(editable);
      setDocumentViewMode("preview");
      setDocumentPanelError("");
    } catch (error) {
      setDocumentPanelError(getErrorMessage(error, "Не удалось загрузить документ"));
      toast.error(getErrorMessage(error, "Не удалось загрузить документ"));
    }
  };

  const handleDeleteDocument = async (doc: DocumentItem) => {
    const documentId = getId(doc as unknown as Record<string, unknown>, ["id", "document_id"]);
    if (!documentId) return;
    if (!confirm("Удалить документ?")) return;

    try {
      await api.delete(`/documents/${documentId}`);
      toast.success("Документ удален");
      setSelectedDocument(null);
      setDocumentVersions([]);
      await fetchRepo();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка удаления документа"));
    }
  };

  const handleSaveDraft = async () => {
    const documentId = getId(selectedDocument, ["id", "document_id"]);
    if (!documentId) return;
    if (!isOwner) return;

    try {
      await api.patch(`/documents/${documentId}/draft`, { content: documentEditorContent });
      toast.success("Черновик сохранен");
      await handleDocumentDetails({ id: documentId });
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка сохранения черновика"));
    }
  };

  const handleCreateDocumentVersion = async () => {
    const documentId = getId(selectedDocument, ["id", "document_id"]);
    if (!documentId) return;
    if (!isOwner) return;
    if (!documentEditorContent.trim()) {
      toast.error("Нельзя создать версию с пустым содержимым");
      return;
    }
    if (!hasDocumentChanges) {
      toast.error("Нет изменений относительно текущей версии");
      return;
    }

    try {
      await api.post(`/documents/${documentId}/versions`, {
        content: documentEditorContent,
        change_summary: documentChangeSummary,
      });
      toast.success("Версия документа создана");
      setDocumentChangeSummary("");
      await handleDocumentDetails({ id: documentId });
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка создания версии документа"));
    }
  };

  const handleRestoreDocumentVersion = async (versionId: string) => {
    const documentId = getId(selectedDocument, ["id", "document_id"]);
    if (!documentId) return;
    if (!isOwner) return;

    try {
      await api.get(`/document-versions/${versionId}`);
      await api.post(`/documents/${documentId}/versions/${versionId}/restore`, {});
      toast.success("Версия документа восстановлена");
      await handleDocumentDetails({ id: documentId });
      await fetchRepo();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка восстановления версии документа"));
    }
  };

  const handleFileDetails = async (file: FileItem) => {
    const fileId = getId(file as unknown as Record<string, unknown>, ["id", "file_id"]);
    if (!fileId) return;

    try {
      const [detailRes, versionsRes] = await Promise.all([
        api.get(`/files/${fileId}`),
        api.get(`/files/${fileId}/versions`),
      ]);
      setSelectedFile((detailRes.data.file || detailRes.data) as Record<string, unknown>);
      setFileVersions((versionsRes.data.versions || []) as VersionItem[]);
      setFilePanelError("");
    } catch (error) {
      setFilePanelError(getErrorMessage(error, "Не удалось загрузить файл"));
      toast.error(getErrorMessage(error, "Не удалось загрузить файл"));
    }
  };

  const handleDeleteFile = async (file: FileItem) => {
    const fileId = getId(file as unknown as Record<string, unknown>, ["id", "file_id"]);
    if (!fileId) return;
    if (!confirm("Удалить файл?")) return;

    try {
      await api.delete(`/files/${fileId}`);
      toast.success("Файл удален");
      setSelectedFile(null);
      setFileVersions([]);
      await fetchRepo();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка удаления файла"));
    }
  };

  const handleAddFileVersion = async () => {
    const fileId = getId(selectedFile, ["id", "file_id"]);
    if (!fileId) return;
    if (!isOwner) return;

    const size = Number.parseInt(fileVersionSize, 10);
    if (!Number.isFinite(size) || size <= 0) {
      toast.error("Размер версии файла должен быть положительным числом");
      return;
    }

    const storageKey = fileVersionStorageKey.trim() || `files/version_${Date.now()}`;

    try {
      await api.post(`/files/${fileId}/versions`, {
        change_summary: fileVersionChangeSummary,
        mime_type: fileVersionMimeType || "application/octet-stream",
        size_bytes: size,
        storage_key: storageKey,
      });
      toast.success("Версия файла добавлена");
      setFileVersionStorageKey("");
      setFileVersionChangeSummary("");
      await handleFileDetails({ id: fileId });
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка добавления версии файла"));
    }
  };

  const handleRestoreFileVersion = async (versionId: string) => {
    const fileId = getId(selectedFile, ["id", "file_id"]);
    if (!fileId) return;
    if (!isOwner) return;

    try {
      await api.get(`/file-versions/${versionId}`);
      await api.post(`/files/${fileId}/versions/${versionId}/restore`, {});
      toast.success("Версия файла восстановлена");
      await handleFileDetails({ id: fileId });
      await fetchRepo();
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка восстановления версии файла"));
    }
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-6 w-full max-w-2xl" />
        <Skeleton className="h-100 w-full" />
      </div>
    );
  }

  if (!repo) {
    if (errorState === "forbidden") {
      return <div className="text-center py-20 text-xl font-bold">Доступ к репозиторию запрещен</div>;
    }
    return <div className="text-center py-20 text-xl font-bold">Репозиторий не найден</div>;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 border-b pb-6 md:flex-row md:items-center md:justify-between">
        <div className="flex min-w-0 flex-wrap items-center gap-2 text-xl sm:text-2xl">
          <Book className="h-6 w-6 text-muted-foreground" />
          <Link href={`/user/${repo.owner_username || owner}`} className="max-w-full text-primary hover:underline cursor-pointer break-all">
            {repo.owner_username || owner}
          </Link>
          <span className="text-muted-foreground">/</span>
          <span className="min-w-0 truncate font-bold">{repo.name}</span>
          <span className="ml-2 px-2 py-0.5 text-xs border rounded-full text-muted-foreground">
            {repo.visibility}
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" onClick={handleToggleStar} disabled={isStarLoading}>
            <Star className="mr-2 h-4 w-4" /> {isStarred ? "Starred" : "Star"} <span className="ml-2 text-muted-foreground">{starsCount}</span>
          </Button>
          <Button variant="outline" size="sm" onClick={openForkModal}>
            <GitFork className="mr-2 h-4 w-4" /> Fork <span className="ml-2 text-muted-foreground">{forks.length}</span>
          </Button>
        </div>
      </div>

      <p className="text-base text-muted-foreground break-words sm:text-lg">{repo.description || "Без описания"}</p>
      {parentRepo ? (
        <p className="text-sm text-muted-foreground break-all">
          Форк от{" "}
          <Link
            href={`/${parentRepo.owner_username || parentRepo.owner_id || "unknown"}/${parentRepo.slug}`}
            className="text-primary hover:underline"
          >
            {parentRepo.owner_username || parentRepo.owner_id || "unknown"}/{parentRepo.slug}
          </Link>
        </p>
      ) : null}

      {/* Tabs */}
      <Tabs defaultValue="documents" className="w-full">
        <TabsList className="mb-4 w-full">
          <TabsTrigger value="documents" className="flex items-center gap-2">
            <FileText className="h-4 w-4" /> Документы
          </TabsTrigger>
          <TabsTrigger value="files" className="flex items-center gap-2">
            <Files className="h-4 w-4" /> Файлы
          </TabsTrigger>
          {isOwner && (
            <TabsTrigger value="settings" className="flex items-center gap-2">
              <Settings className="h-4 w-4" /> Настройки
            </TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="documents" className="space-y-4">
          <div className="border rounded-md overflow-hidden">
            <div className="bg-muted border-b px-4 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <span className="font-medium text-sm">Документы в репозитории</span>
              {isOwner ? <Button size="sm" className="w-full sm:w-auto" onClick={() => setCreateDocumentModalOpen(true)}>Создать документ</Button> : null}
            </div>
            {documents.length > 0 ? (
              <div className="divide-y">
                {documents.map((doc) => (
                  <div key={doc.id || doc.document_id || doc.title} className="p-4 flex flex-col gap-3 hover:bg-muted/50 transition sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex min-w-0 items-center gap-3">
                      <FileText className="h-5 w-5 text-primary" />
                      <span
                        className="block min-w-0 cursor-pointer truncate font-medium hover:underline"
                        onClick={() => {
                          const documentId = getId(doc as unknown as Record<string, unknown>, ["id", "document_id"]);
                          if (!documentId) return;
                          router.push(`/${owner}/${slug}/blob/document/${documentId}`);
                        }}
                      >
                        {doc.title || "Без названия"}
                      </span>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground sm:justify-end">
                      <span className="capitalize">{doc.status || "draft"}</span>
                      <span>{doc.updated_at ? new Date(doc.updated_at).toLocaleDateString("ru-RU") : "-"}</span>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          const documentId = getId(doc as unknown as Record<string, unknown>, ["id", "document_id"]);
                          if (!documentId) return;
                          router.push(`/${owner}/${slug}/blob/document/${documentId}`);
                        }}
                      >
                        Открыть
                      </Button>
                      {isOwner ? <Button variant="destructive" size="sm" onClick={() => void handleDeleteDocument(doc)}>Удалить</Button> : null}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <SectionState title="Пока нет документов" description="Создайте первый документ, чтобы начать вести материалы." />
            )}
          </div>
          <SectionState title="Просмотр на отдельной странице" description="Откройте документ, чтобы перейти на отдельную страницу просмотра и редактирования как в GitHub." />
        </TabsContent>

        <TabsContent value="files">
          <div className="border rounded-md overflow-hidden">
            <div className="bg-muted border-b px-4 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <span className="font-medium text-sm">Файлы (бинарные архивы, PDF и пр.)</span>
              {isOwner ? <Button size="sm" className="w-full sm:w-auto" onClick={() => setCreateFileModalOpen(true)}>Загрузить файл</Button> : null}
            </div>
            {files.length > 0 ? (
              <div className="divide-y">
                {files.map((file) => (
                  <div key={file.id || file.file_id || file.file_name} className="p-4 flex flex-col gap-3 hover:bg-muted/50 transition sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex min-w-0 items-center gap-3">
                      <Files className="h-5 w-5 text-primary" />
                      <span
                        className="block min-w-0 cursor-pointer truncate font-medium hover:underline"
                        onClick={() => {
                          const fileId = getId(file as unknown as Record<string, unknown>, ["id", "file_id"]);
                          if (!fileId) return;
                          router.push(`/${owner}/${slug}/blob/file/${fileId}`);
                        }}
                      >
                        {file.file_name || "Без имени"}
                      </span>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground sm:justify-end">
                      <span>{file.mime_type || "application/octet-stream"}</span>
                      <span>{Math.round(((file.size_bytes || file.size || 0) as number) / 1024)} KB</span>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          const fileId = getId(file as unknown as Record<string, unknown>, ["id", "file_id"]);
                          if (!fileId) return;
                          router.push(`/${owner}/${slug}/blob/file/${fileId}`);
                        }}
                      >
                        Открыть
                      </Button>
                      {isOwner ? <Button variant="destructive" size="sm" onClick={() => void handleDeleteFile(file)}>Удалить</Button> : null}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <SectionState title="Файлы отсутствуют" description="Добавьте первый файл в репозиторий." />
            )}
          </div>
          <SectionState title="Просмотр на отдельной странице" description="Откройте файл, чтобы перейти на отдельную страницу с версионностью и метаданными." />
        </TabsContent>

        {isOwner && (
          <TabsContent value="settings">
            <div className="p-4 border rounded-md space-y-4">
              <h3 className="font-bold text-lg mb-2">Настройки репозитория</h3>
              <p className="text-muted-foreground mb-4">Настройки видимости, переименование, удаление.</p>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button variant="outline" className="w-full sm:w-auto" onClick={() => setEditRepoModalOpen(true)}>Редактировать репозиторий</Button>
                <Button variant="destructive" className="w-full sm:w-auto" onClick={handleDelete}>Удалить репозиторий</Button>
              </div>

              <div className="border-t pt-4 space-y-3">
                <h4 className="font-medium">Форки ({forks.length})</h4>
                {forks.length === 0 ? (
                  <p className="text-sm text-muted-foreground">Пока нет форков.</p>
                ) : (
                  <div className="space-y-2">
                    {forks.map((fork) => (
                      <div key={fork.id || fork.repo_id || fork.slug} className="text-sm text-muted-foreground">
                        <button
                          type="button"
                          className="text-primary hover:underline"
                          onClick={() => {
                            const forkOwner = fork.owner_username || fork.owner_id;
                            if (!forkOwner || !fork.slug) return;
                            router.push(`/${forkOwner}/${fork.slug}`);
                          }}
                        >
                          {fork.owner_username || fork.owner_id || "unknown"}/{fork.slug}
                        </button>{" "}
                        - {fork.name}
                      </div>
                    ))}
                  </div>
                )}
              </div>

            </div>
          </TabsContent>
        )}
      </Tabs>

      <Modal
        open={isForkModalOpen}
        title="Создать форк"
        onClose={() => {
          if (!isForking) {
            setForkModalOpen(false);
          }
        }}
        footer={
          <>
            <Button variant="outline" disabled={isForking} onClick={() => setForkModalOpen(false)}>Отмена</Button>
            <Button disabled={isForking || !forkVisibility} onClick={handleFork}>
              {isForking ? "Создание..." : "Создать форк"}
            </Button>
          </>
        }
      >
        <p className="text-sm text-muted-foreground">
          Будет создан форк из {owner}/{repo.slug}. Slug создается автоматически на основе названия.
        </p>
        <div className="space-y-2">
          <p className="text-sm font-medium">Видимость форка</p>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button
              type="button"
              variant={forkVisibility === "public" ? "default" : "outline"}
              className="w-full sm:w-auto"
              onClick={() => setForkVisibility("public")}
            >
              Public
            </Button>
            <Button
              type="button"
              variant={forkVisibility === "private" ? "default" : "outline"}
              className="w-full sm:w-auto"
              onClick={() => setForkVisibility("private")}
            >
              Private
            </Button>
          </div>
        </div>
        <div className="rounded-md border bg-muted/30 px-3 py-2 text-sm">
          <p className="text-muted-foreground">Будущий путь</p>
          <p className="font-medium text-primary">{forkOwner}/{effectiveForkSlug}</p>
        </div>
        <Input
          value={forkName}
          onChange={(e) => setForkName(e.target.value)}
          placeholder="Название форка (опционально)"
        />
        <Textarea
          value={forkDescription}
          onChange={(e) => setForkDescription(e.target.value)}
          placeholder="Описание форка (опционально)"
          rows={4}
        />
      </Modal>

      {isOwner && (
        <>
          <Modal
            open={isCreateDocumentModalOpen}
            title="Создать документ"
            onClose={() => setCreateDocumentModalOpen(false)}
            footer={
              <>
                <Button variant="outline" onClick={() => setCreateDocumentModalOpen(false)}>Отмена</Button>
                <Button onClick={handleCreateDocument}>Создать</Button>
              </>
            }
          >
            <Input
              value={newDocTitle}
              onChange={(e) => setNewDocTitle(e.target.value)}
              placeholder="Название документа"
            />
            <Textarea
              value={newDocContent}
              onChange={(e) => setNewDocContent(e.target.value)}
              placeholder="Первичное содержимое"
              rows={8}
            />
          </Modal>

          <Modal
            open={isCreateFileModalOpen}
            title="Загрузить файл"
            onClose={() => {
              setCreateFileModalOpen(false);
              setSelectedUploadFile(null);
            }}
            footer={
              <>
                <Button
                  variant="outline"
                  onClick={() => {
                    setCreateFileModalOpen(false);
                    setSelectedUploadFile(null);
                  }}
                >
                  Отмена
                </Button>
                <Button onClick={handleAddFile}>Добавить</Button>
              </>
            }
          >
            <Input
              type="file"
              onChange={(e) => handleSelectUploadFile(e.target.files?.[0] || null)}
            />
            <p className="text-xs text-muted-foreground">
              Выберите локальный файл для автозаполнения полей. На текущем backend выполняется регистрация файла по метаданным.
            </p>
            <Input
              value={newFileName}
              onChange={(e) => setNewFileName(e.target.value)}
              placeholder="Имя файла"
            />
            {selectedUploadFile ? (
              <p className="text-xs text-muted-foreground">
                Выбран файл: {selectedUploadFile.name} ({Math.round(selectedUploadFile.size / 1024)} KB)
              </p>
            ) : null}
          </Modal>

          <Modal
            open={isEditRepoModalOpen}
            title="Редактировать репозиторий"
            onClose={() => setEditRepoModalOpen(false)}
            footer={
              <>
                <Button variant="outline" onClick={() => setEditRepoModalOpen(false)}>Отмена</Button>
                <Button onClick={handleEditRepo}>Сохранить</Button>
              </>
            }
          >
            <Input
              value={editRepoName}
              onChange={(e) => setEditRepoName(e.target.value)}
              placeholder="Название репозитория"
            />
            <Textarea
              value={editRepoDescription}
              onChange={(e) => setEditRepoDescription(e.target.value)}
              placeholder="Описание репозитория"
              rows={5}
            />
          </Modal>
        </>
      )}
    </div>
  );
}
