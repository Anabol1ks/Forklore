"use client";

import { Suspense, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import axios from "axios";
import { api } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { Search, BookOpen, FileText, Files, ArrowUpRight, Sparkles } from "lucide-react";

type SearchEntityType = "all" | "repository" | "document" | "file";

interface SearchHit {
  entity_type: "repository" | "document" | "file" | "unspecified";
  entity_id: string;
  repo_id?: string | null;
  owner_id?: string | null;
  tag_id?: string | null;
  title: string;
  description?: string | null;
  snippet?: string | null;
  rank: number;
  updated_at: string;
}

interface SearchResponse {
  hits: SearchHit[];
  total: number;
}

interface RepositoryResolve {
  owner_id?: string;
  slug?: string;
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error)) {
    return (error.response?.data as { message?: string } | undefined)?.message || fallback;
  }
  return fallback;
}

function typeLabel(type: SearchEntityType): string {
  switch (type) {
    case "repository":
      return "Repositories";
    case "document":
      return "Documents";
    case "file":
      return "Files";
    default:
      return "All";
  }
}

function iconByType(type: SearchHit["entity_type"]) {
  switch (type) {
    case "repository":
      return <BookOpen className="h-4 w-4" />;
    case "document":
      return <FileText className="h-4 w-4" />;
    case "file":
      return <Files className="h-4 w-4" />;
    default:
      return <Search className="h-4 w-4" />;
  }
}

function isUUID(value: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value);
}

function parseOwnerAndSlugFromTitle(title: string): { owner: string; slug: string } | null {
  const value = title.trim();
  const slashIndex = value.indexOf("/");
  if (slashIndex <= 0 || slashIndex >= value.length - 1) {
    return null;
  }

  const owner = value.slice(0, slashIndex).trim();
  const slug = value.slice(slashIndex + 1).trim();
  if (!owner || !slug) {
    return null;
  }

  return { owner, slug };
}

function stripKnownTags(value: string): string {
  return value.replace(/<\/?b>/gi, "").trim();
}

function renderSnippet(snippet: string | null | undefined): ReactNode {
  const source = (snippet || "").trim();
  if (!source) {
    return "Без описания";
  }

  const parts = source.split(/(<b>.*?<\/b>)/gi).filter(Boolean);
  return parts.map((part, index) => {
    const boldMatch = part.match(/^<b>(.*?)<\/b>$/i);
    if (boldMatch) {
      return (
        <mark key={`snippet-mark-${index}`} className="px-1 rounded bg-primary/15 text-foreground">
          {stripKnownTags(boldMatch[1] || "")}
        </mark>
      );
    }

    return <span key={`snippet-text-${index}`}>{stripKnownTags(part)}</span>;
  });
}

function SearchPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { user } = useAuthStore();

  const initialQuery = searchParams.get("q") || "";
  const initialType = (searchParams.get("type") || "all") as SearchEntityType;

  const [queryInput, setQueryInput] = useState(initialQuery);
  const [activeType, setActiveType] = useState<SearchEntityType>(
    initialType === "repository" || initialType === "document" || initialType === "file" ? initialType : "all"
  );
  const [hits, setHits] = useState<SearchHit[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  const repoCacheRef = useRef<Map<string, RepositoryResolve>>(new Map());

  const pageSize = 20;
  const hasMore = hits.length < total;
  const normalizedQuery = queryInput.trim();

  const typeFilters: SearchEntityType[] = useMemo(() => ["all", "repository", "document", "file"], []);

  const updateUrl = useCallback((nextQuery: string, nextType: SearchEntityType) => {
    const params = new URLSearchParams();
    if (nextQuery.trim()) {
      params.set("q", nextQuery.trim());
    }
    if (nextType !== "all") {
      params.set("type", nextType);
    }

    const next = params.toString();
    router.replace(next ? `/search?${next}` : "/search");
  }, [router]);

  const performSearch = useCallback(async (offset: number, append: boolean) => {
    const q = queryInput.trim();
    if (!q) {
      setHits([]);
      setTotal(0);
      return;
    }

    if (append) {
      setLoadingMore(true);
    } else {
      setLoading(true);
    }

    try {
      const payload: {
        query: string;
        entity_types?: string[];
        limit: number;
        offset: number;
      } = {
        query: q,
        limit: pageSize,
        offset,
      };

      if (activeType !== "all") {
        payload.entity_types = [activeType];
      }

      const response = await api.post<SearchResponse>("/search", payload);
      const nextHits = response.data.hits || [];
      setTotal(response.data.total || 0);
      setHits((prev) => (append ? [...prev, ...nextHits] : nextHits));
    } catch (error) {
      toast.error(getErrorMessage(error, "Не удалось выполнить поиск"));
      if (!append) {
        setHits([]);
        setTotal(0);
      }
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [activeType, queryInput]);

  useEffect(() => {
    const urlQuery = searchParams.get("q") || "";
    const urlType = (searchParams.get("type") || "all") as SearchEntityType;
    setQueryInput(urlQuery);
    setActiveType(urlType === "repository" || urlType === "document" || urlType === "file" ? urlType : "all");
  }, [searchParams]);

  useEffect(() => {
    if (!normalizedQuery) {
      setHits([]);
      setTotal(0);
      return;
    }
    void performSearch(0, false);
  }, [activeType, normalizedQuery, performSearch]);

  const handleSubmit = () => {
    updateUrl(queryInput, activeType);
  };

  const resolveRepository = async (repoId: string): Promise<RepositoryResolve | null> => {
    const cached = repoCacheRef.current.get(repoId);
    if (cached) return cached;

    try {
      const response = await api.get(`/repositories/${repoId}`);
      const repo = (response.data.repository || response.data) as RepositoryResolve;
      const resolved = {
        owner_id: repo.owner_id,
        slug: repo.slug,
      };
      repoCacheRef.current.set(repoId, resolved);
      return resolved;
    } catch {
      return null;
    }
  };

  const openHit = async (hit: SearchHit) => {
    const entityType = hit.entity_type;
    if (entityType === "unspecified") {
      toast.error("Неизвестный тип результата");
      return;
    }

    const repoId = entityType === "repository" ? hit.entity_id : hit.repo_id || "";
    if (!repoId) {
      toast.error("Невозможно определить репозиторий результата");
      return;
    }

    const repo = await resolveRepository(repoId);
    const parsedRepoPath = parseOwnerAndSlugFromTitle(hit.title || "");

    let owner = parsedRepoPath?.owner || repo?.owner_id || hit.owner_id || "";
    const slug = parsedRepoPath?.slug || repo?.slug || "";

    if (user?.id && owner === user.id && user.username) {
      owner = user.username;
    }

    if (!owner || !slug) {
      toast.error("Не удалось открыть результат поиска");
      return;
    }

    if (entityType === "repository") {
      router.push(`/${owner}/${slug}`);
      return;
    }

    router.push(`/${owner}/${slug}/blob/${entityType}/${hit.entity_id}`);
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6">
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-xl font-semibold">
          <Sparkles className="h-5 w-5 text-primary" />
          <span>Global Search</span>
        </div>
        <p className="text-sm text-muted-foreground">
          Поиск по репозиториям, документам и файлам. Формат и поведение близки к GitHub: единая строка, фильтр по типам, быстрый переход.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6 space-y-4">
          <div className="relative">
            <Search className="h-4 w-4 text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2" />
            <Input
              value={queryInput}
              onChange={(e) => setQueryInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  handleSubmit();
                }
              }}
              className="pl-9 h-11"
              placeholder="Search repositories, documents, files"
            />
          </div>

          <div className="flex flex-wrap gap-2">
            {typeFilters.map((type) => (
              <Button
                key={type}
                size="sm"
                variant={activeType === type ? "default" : "outline"}
                onClick={() => {
                  setActiveType(type);
                  updateUrl(queryInput, type);
                }}
              >
                {typeLabel(type)}
              </Button>
            ))}
            <Button size="sm" className="ml-auto" onClick={handleSubmit}>Search</Button>
          </div>
        </CardContent>
      </Card>

      {!normalizedQuery ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            Введите запрос для поиска.
          </CardContent>
        </Card>
      ) : null}

      {loading ? (
        <div className="space-y-3">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
        </div>
      ) : null}

      {!loading && normalizedQuery ? (
        <div className="space-y-3">
          <div className="text-sm text-muted-foreground">
            Найдено: {total}
          </div>

          {hits.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                Ничего не найдено.
              </CardContent>
            </Card>
          ) : (
            hits.map((hit) => (
              <Card key={`${hit.entity_type}-${hit.entity_id}`} className="hover:border-primary/40 transition-colors">
                <CardHeader className="pb-3">
                  <CardTitle className="text-base flex items-center justify-between gap-2">
                    <span className="flex items-center gap-2">
                      {iconByType(hit.entity_type)}
                      {hit.title || "Без названия"}
                    </span>
                    <span className="text-xs font-normal text-muted-foreground capitalize">
                      {hit.entity_type}
                    </span>
                  </CardTitle>
                  <CardDescription className="line-clamp-2">
                    {renderSnippet(hit.snippet || hit.description)}
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-0 flex items-center justify-between gap-4">
                  <div className="text-xs text-muted-foreground">
                    {hit.updated_at ? new Date(hit.updated_at).toLocaleString("ru-RU") : ""}
                  </div>
                  <Button size="sm" variant="outline" onClick={() => void openHit(hit)}>
                    Открыть <ArrowUpRight className="h-4 w-4 ml-1" />
                  </Button>
                </CardContent>
              </Card>
            ))
          )}

          {hasMore ? (
            <div className="flex justify-center pt-2">
              <Button
                variant="outline"
                onClick={() => void performSearch(hits.length, true)}
                disabled={loadingMore}
              >
                {loadingMore ? "Загрузка..." : "Показать ещё"}
              </Button>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense
      fallback={
        <div className="max-w-5xl mx-auto space-y-3">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-full" />
        </div>
      }
    >
      <SearchPageContent />
    </Suspense>
  );
}
