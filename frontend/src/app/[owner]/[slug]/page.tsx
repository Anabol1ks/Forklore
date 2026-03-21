
"use client";

import { useRepo } from "@/components/repo/repo-context";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Book, FileText, Files } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";

interface DocumentItem {
  id?: string;
  document_id?: string;
  title?: string;
  slug?: string;
}

interface FileItem {
  id?: string;
  file_id?: string;
  file_name?: string;
  size_bytes?: number;
}

function toSlugSegment(value: string): string {
  return value
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9а-яё-]/gi, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function RepositoryFileSystemPage({ params }: { params: { owner: string; slug: string } }) {
  const { repo, loading, errorState, repoId, isOwner } = useRepo();
  const [documents, setDocuments] = useState<DocumentItem[]>([]);
  const [files, setFiles] = useState<FileItem[]>([]);
  const [itemsLoading, setItemsLoading] = useState(true);

  useEffect(() => {
    async function fetchItems() {
      if (!repoId) return;
      try {
        setItemsLoading(true);
        const [docsRes, filesRes] = await Promise.all([
          api.get(`/repositories/${repoId}/documents`),
          api.get(`/repositories/${repoId}/files`),
        ]);
        setDocuments((docsRes.data.documents || []) as DocumentItem[]);
        setFiles((filesRes.data.files || []) as FileItem[]);
      } catch (error) {
        console.error("Failed to fetch repository items", error);
      } finally {
        setItemsLoading(false);
      }
    }
    
    void fetchItems();
  }, [repoId]);

  if (errorState === "not-found") {
    return <div className="text-center py-20 text-xl font-bold">Репозиторий не найден</div>;
  }
  if (errorState === "forbidden") {
    return <div className="text-center py-20 text-xl font-bold">Нет доступа к репозиторию</div>;
  }

  if (loading || itemsLoading) {
    return (
      <div className="space-y-4 mt-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (!repo) return null;

  return (
    <div className="mt-4">
      {isOwner ? (
        <div className="mb-3 flex justify-end gap-2">
          <Link href={`/${params.owner}/${params.slug}/new-document`}>
            <Button variant="outline" size="sm">Новый документ</Button>
          </Link>
          <Link href={`/${params.owner}/${params.slug}/upload-file`}>
            <Button size="sm">Загрузить файл</Button>
          </Link>
        </div>
      ) : null}

      <div className="border rounded-md shadow-sm">
        <div className="bg-muted/40 p-3 border-b flex items-center gap-2 text-sm font-medium">
          <Book className="h-4 w-4" /> Корневая директория
        </div>
        
        {documents.length === 0 && files.length === 0 ? (
          <div className="p-8 text-center text-muted-foreground">
            Репозиторий пуст.
          </div>
        ) : (
          <div className="divide-y">
            {documents.map((doc) => {
              const docId = doc.id || doc.document_id;
              const routeKey = (doc.slug && doc.slug.trim()) || (doc.title ? toSlugSegment(doc.title) : "") || docId;
              return (
                <Link
                  key={docId}
                  href={`/${params.owner}/${params.slug}/blob/document/${encodeURIComponent(String(routeKey || ""))}`}
                  className="flex items-center gap-2 p-3 hover:bg-muted/50 transition-colors"
                >
                  <FileText className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-primary hover:underline">{doc.title || "Документ"}</span>
                </Link>
              );
            })}

            {files.map((file) => {
              const fileId = file.id || file.file_id;
              return (
                <Link
                  key={fileId}
                  href={`/${params.owner}/${params.slug}/blob/file/${encodeURIComponent(String(fileId || ""))}`}
                  className="flex items-center gap-2 p-3 hover:bg-muted/50 transition-colors"
                >
                  <Files className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-primary hover:underline">{file.file_name || "Файл"}</span>
                  <span className="text-xs text-muted-foreground ml-auto">{file.size_bytes ? `${Math.round(file.size_bytes / 1024)} KB` : ""}</span>
                </Link>
              );
            })}
          </div>
        )}
      </div>
      
      {repo.description && (
        <div className="mt-6 border rounded-md p-4">
          <h2 className="text-sm font-medium mb-2">Об этом репозитории</h2>
          <p className="text-sm text-muted-foreground whitespace-pre-wrap">{repo.description}</p>
        </div>
      )}
    </div>
  );
}

