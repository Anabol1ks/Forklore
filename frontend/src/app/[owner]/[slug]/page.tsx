"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Book, FileText, GitFork, Settings, Star, Files } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { useAuthStore } from "@/store/auth";
import axios from "axios";

interface Repository {
  id?: string;
  repo_id?: string;
  name: string;
  description?: string;
  visibility: string;
  owner_id?: string;
  slug: string;
  created_at?: string;
}

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

export default function RepositoryPage() {
  const params = useParams();
  const router = useRouter();
  const owner = params.owner as string;
  const slug = params.slug as string;
  const { user } = useAuthStore();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorState, setErrorState] = useState<"not-found" | "forbidden" | null>(null);

  const [documents, setDocuments] = useState<DocumentItem[]>([]);
  const [files, setFiles] = useState<FileItem[]>([]);
  const [forks, setForks] = useState<Repository[]>([]);

  const [selectedDocument, setSelectedDocument] = useState<Record<string, unknown> | null>(null);
  const [selectedFile, setSelectedFile] = useState<Record<string, unknown> | null>(null);
  const [documentVersions, setDocumentVersions] = useState<VersionItem[]>([]);
  const [fileVersions, setFileVersions] = useState<VersionItem[]>([]);
  const isOwner = !!user?.id && !!repo?.owner_id && user.id === repo.owner_id;

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

      setRepo((repoByIdResponse.data.repository || repoData) as Repository);
      setDocuments((docsRes.data.documents || []) as DocumentItem[]);
      setFiles((filesRes.data.files || []) as FileItem[]);
      setForks((forksRes.data.repositories || []) as Repository[]);
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

  const handleFork = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    const newName = prompt("Имя форка (можно оставить пустым):") || "";
    const newSlug = prompt("Slug форка (можно оставить пустым):") || "";

    try {
      const response = await api.post(`/repositories/${repoId}/fork`, {
        name: newName,
        slug: newSlug,
      });
      toast.success("Репозиторий успешно форкнут!");
      const newRepo = (response.data.repository || response.data) as Repository;
      if (user?.username && newRepo.slug) {
        router.push(`/${user.username}/${newRepo.slug}`);
      }
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка при форке репозитория"));
    }
  };

  const handleAddFile = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    const name = prompt("Введите имя файла (тестовое добавление):");
    if (!name) return;
    const mimeType = prompt("MIME type файла:", "application/octet-stream") || "application/octet-stream";

    try {
      await api.post(`/repositories/${repoId}/files`, {
        file_name: name,
        storage_key: `files/${Date.now()}_${name}`,
        size_bytes: 1024,
        mime_type: mimeType,
      });
      toast.success("Файл добавлен!");
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка добавления файла"));
    }
  };

  const handleEditRepo = async () => {
    if (!repo) return;
    const repoId = getId(repo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) return;

    const newName = prompt("Новое имя репозитория:", repo.name);
    if (!newName) return;
    const newDesc = prompt("Новое описание:", repo.description || "") || "";

    try {
      await api.patch(`/repositories/${repoId}`, { name: newName, description: newDesc });
      toast.success("Репозиторий обновлен!");
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

    const title = prompt("Введите название документа:");
    if (!title) return;
    const content = prompt("Первичное содержимое (необязательно):") || "";

    try {
      await api.post(`/repositories/${repoId}/documents`, {
        title,
        initial_content: content,
      });
      toast.success("Документ создан!");
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
      setSelectedDocument((detailRes.data.document || detailRes.data) as Record<string, unknown>);
      setDocumentVersions((versionsRes.data.versions || []) as VersionItem[]);
    } catch (error) {
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
    const content = prompt("Текст черновика:");
    if (content === null) return;

    try {
      await api.patch(`/documents/${documentId}/draft`, { content });
      toast.success("Черновик сохранен");
      await handleDocumentDetails({ id: documentId });
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка сохранения черновика"));
    }
  };

  const handleCreateDocumentVersion = async () => {
    const documentId = getId(selectedDocument, ["id", "document_id"]);
    if (!documentId) return;
    const content = prompt("Содержимое новой версии:");
    if (!content) return;
    const changeSummary = prompt("Описание изменений:") || "";

    try {
      await api.post(`/documents/${documentId}/versions`, {
        content,
        change_summary: changeSummary,
      });
      toast.success("Версия документа создана");
      await handleDocumentDetails({ id: documentId });
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка создания версии документа"));
    }
  };

  const handleRestoreDocumentVersion = async () => {
    const documentId = getId(selectedDocument, ["id", "document_id"]);
    if (!documentId) return;
    const versionId = prompt("ID версии для восстановления:");
    if (!versionId) return;

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
    } catch (error) {
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

    const changeSummary = prompt("Описание изменений версии файла:") || "";
    const mimeType = prompt("MIME type версии:", "application/octet-stream") || "application/octet-stream";

    try {
      await api.post(`/files/${fileId}/versions`, {
        change_summary: changeSummary,
        mime_type: mimeType,
        size_bytes: 1024,
        storage_key: `files/version_${Date.now()}`,
      });
      toast.success("Версия файла добавлена");
      await handleFileDetails({ id: fileId });
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка добавления версии файла"));
    }
  };

  const handleRestoreFileVersion = async () => {
    const fileId = getId(selectedFile, ["id", "file_id"]);
    if (!fileId) return;
    const versionId = prompt("ID версии файла для восстановления:");
    if (!versionId) return;

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
        <Skeleton className="h-10 w-75" />
        <Skeleton className="h-6 w-125" />
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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b pb-6">
        <div className="flex items-center gap-2 text-2xl">
          <Book className="h-6 w-6 text-muted-foreground" />
          <span className="text-primary hover:underline cursor-pointer">{owner}</span>
          <span className="text-muted-foreground">/</span>
          <span className="font-bold">{repo.name}</span>
          <span className="ml-2 px-2 py-0.5 text-xs border rounded-full text-muted-foreground">
            {repo.visibility}
          </span>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm">
            <Star className="mr-2 h-4 w-4" /> Star <span className="ml-2 text-muted-foreground">0</span>
          </Button>
          <Button variant="outline" size="sm" onClick={handleFork}>
            <GitFork className="mr-2 h-4 w-4" /> Fork <span className="ml-2 text-muted-foreground">{forks.length}</span>
          </Button>
        </div>
      </div>

      <p className="text-lg text-muted-foreground">{repo.description || "Без описания"}</p>

      {/* Tabs */}
      <Tabs defaultValue="documents" className="w-full">
        <TabsList className="mb-4">
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
            <div className="bg-muted px-4 py-3 border-b flex justify-between items-center">
              <span className="font-medium text-sm">Документы в репозитории</span>
              {isOwner ? <Button size="sm" onClick={handleCreateDocument}>Создать документ</Button> : null}
            </div>
            {documents.length > 0 ? (
               <div className="divide-y">
                 {documents.map((doc) => (
                   <div key={doc.id || doc.document_id || doc.title} className="p-4 flex items-center justify-between hover:bg-muted/50 transition">
                     <div className="flex items-center gap-3">
                       <FileText className="h-5 w-5 text-primary" />
                       <span className="font-medium cursor-pointer hover:underline" onClick={() => void handleDocumentDetails(doc)}>{doc.title || "Без названия"}</span>
                     </div>
                     <div className="text-sm text-muted-foreground flex gap-3 items-center">
                       <span className="capitalize">{doc.status || "draft"}</span>
                       <span>{doc.updated_at ? new Date(doc.updated_at).toLocaleDateString("ru-RU") : "-"}</span>
                       <Button variant="outline" size="sm" onClick={() => void handleDocumentDetails(doc)}>Детали</Button>
                       {isOwner ? <Button variant="destructive" size="sm" onClick={() => void handleDeleteDocument(doc)}>Удалить</Button> : null}
                     </div>
                   </div>
                 ))}
               </div>
            ) : (
                <div className="p-8 text-center text-muted-foreground flex flex-col items-center">
                <FileText className="h-10 w-10 mb-2 opacity-20" />
                <p>Пока нет документов.</p>
                </div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="files">
          <div className="border rounded-md overflow-hidden">
            <div className="bg-muted px-4 py-3 border-b flex justify-between items-center">
              <span className="font-medium text-sm">Файлы (бинарные архивы, PDF и пр.)</span>
              {isOwner ? <Button size="sm" onClick={handleAddFile}>Загрузить файл</Button> : null}
            </div>
            {files.length > 0 ? (
               <div className="divide-y">
                 {files.map((file) => (
                   <div key={file.id || file.file_id || file.file_name} className="p-4 flex items-center justify-between hover:bg-muted/50 transition">
                     <div className="flex items-center gap-3">
                       <Files className="h-5 w-5 text-primary" />
                       <span className="font-medium cursor-pointer hover:underline" onClick={() => void handleFileDetails(file)}>{file.file_name || "Без имени"}</span>
                     </div>
                     <div className="text-sm text-muted-foreground flex gap-3 items-center">
                       <span>{file.mime_type || "application/octet-stream"}</span>
                       <span>{Math.round(((file.size_bytes || file.size || 0) as number) / 1024)} KB</span>
                       <Button variant="outline" size="sm" onClick={() => void handleFileDetails(file)}>Детали</Button>
                       {isOwner ? <Button variant="destructive" size="sm" onClick={() => void handleDeleteFile(file)}>Удалить</Button> : null}
                     </div>
                   </div>
                 ))}
               </div>
            ) : (
                <div className="p-8 text-center text-muted-foreground flex flex-col items-center">
                <Files className="h-10 w-10 mb-2 opacity-20" />
                <p>Файлы отсутствуют.</p>
                </div>
            )}
          </div>
        </TabsContent>

        {isOwner && (
          <TabsContent value="settings">
            <div className="p-4 border rounded-md space-y-4">
              <h3 className="font-bold text-lg mb-2">Настройки репозитория</h3>
              <p className="text-muted-foreground mb-4">Настройки видимости, переименование, удаление.</p>
              <div className="flex gap-4">
                <Button variant="outline" onClick={handleEditRepo}>Редактировать репозиторий</Button>
                <Button variant="destructive" onClick={handleDelete}>Удалить репозиторий</Button>
              </div>

              <div className="border-t pt-4 space-y-3">
                <h4 className="font-medium">Форки ({forks.length})</h4>
                {forks.length === 0 ? (
                  <p className="text-sm text-muted-foreground">Пока нет форков.</p>
                ) : (
                  <div className="space-y-2">
                    {forks.map((fork) => (
                      <div key={fork.id || fork.repo_id || fork.slug} className="text-sm text-muted-foreground">
                        {fork.name} ({fork.slug})
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {selectedDocument && (
                <div className="border-t pt-4 space-y-3">
                  <h4 className="font-medium">Документ: {String(selectedDocument.title || selectedDocument.slug || "-")}</h4>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={handleSaveDraft}>Сохранить черновик</Button>
                    <Button variant="outline" size="sm" onClick={handleCreateDocumentVersion}>Создать версию</Button>
                    <Button variant="outline" size="sm" onClick={handleRestoreDocumentVersion}>Восстановить версию</Button>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    Версий документа: {documentVersions.length}
                  </div>
                </div>
              )}

              {selectedFile && (
                <div className="border-t pt-4 space-y-3">
                  <h4 className="font-medium">Файл: {String(selectedFile.file_name || "-")}</h4>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" onClick={handleAddFileVersion}>Добавить версию файла</Button>
                    <Button variant="outline" size="sm" onClick={handleRestoreFileVersion}>Восстановить версию файла</Button>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    Версий файла: {fileVersions.length}
                  </div>
                </div>
              )}
            </div>
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}
