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
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "sonner";
import { Search, BookOpen, FileText, Files, ArrowUpRight, Sparkles } from "lucide-react";

type SearchEntityType = "all" | "repository" | "document" | "file";
type SearchMode = "content" | "users";

type RepositoryTag = {
  tag_id: string;
  name: string;
};

type UserSearchResult = {
  user_id: string;
  username: string;
  display_name?: string;
  avatar_url?: string;
  university?: string;
  repositories_count: number;
};

interface SearchUsersResponse {
  users: UserSearchResult[];
  total: number;
}

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
  owner_username?: string;
  slug?: string;
}

const UNIVERSITY_OPTIONS = ["", "МИРЭА", "МГУ"] as const;

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

function formatHitTitle(hit: SearchHit, resolvedRepo?: RepositoryResolve | null): string {
  const fallback = hit.title || "Без названия";
  const parsed = parseOwnerAndSlugFromTitle(fallback);

  if (!parsed) {
    return fallback;
  }

  const owner = isUUID(parsed.owner) ? (resolvedRepo?.owner_username || parsed.owner) : parsed.owner;
  const slug = resolvedRepo?.slug || parsed.slug;
  return `${owner}/${slug}`;
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
  const initialMode = (searchParams.get("mode") || "content") as SearchMode;
  const initialUniversity = searchParams.get("university") || "";
  const initialSubjectTagID = searchParams.get("subject_tag_id") || "";

  const [queryInput, setQueryInput] = useState(initialQuery);
  const [activeMode, setActiveMode] = useState<SearchMode>(initialMode === "users" ? "users" : "content");
  const [activeType, setActiveType] = useState<SearchEntityType>(
    initialType === "repository" || initialType === "document" || initialType === "file" ? initialType : "all"
  );
  const [selectedUniversity, setSelectedUniversity] = useState(initialUniversity);
  const [selectedSubjectTagID, setSelectedSubjectTagID] = useState(initialSubjectTagID);
  const [tags, setTags] = useState<RepositoryTag[]>([]);
  const [hits, setHits] = useState<SearchHit[]>([]);
  const [userHits, setUserHits] = useState<UserSearchResult[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [, setRepoLookupTick] = useState(0);

  const repoCacheRef = useRef<Map<string, RepositoryResolve>>(new Map());

  const pageSize = 20;
  const hasMore = activeMode === "content" && hits.length < total;
  const normalizedQuery = queryInput.trim();

  const typeFilters: SearchEntityType[] = useMemo(() => ["all", "repository", "document", "file"], []);
  const modes: SearchMode[] = useMemo(() => ["content", "users"], []);

  const subjectOptions = useMemo(() => {
    return tags.filter((tag) => {
      if (selectedUniversity === "МИРЭА") {
        return tag.name.startsWith("МИРЭА • ");
      }
      if (selectedUniversity === "МГУ") {
        return tag.name.startsWith("МГУ • ");
      }
      return false;
    });
  }, [selectedUniversity, tags]);

  const selectedSubject = useMemo(() => subjectOptions.find((item) => item.tag_id === selectedSubjectTagID), [selectedSubjectTagID, subjectOptions]);

  const updateUrl = useCallback((nextQuery: string, nextType: SearchEntityType, nextMode: SearchMode, nextUniversity: string, nextSubjectTagID: string) => {
    const params = new URLSearchParams();
    if (nextQuery.trim()) {
      params.set("q", nextQuery.trim());
    }
    if (nextMode === "content" && nextType !== "all") {
      params.set("type", nextType);
    }
    if (nextMode !== "content") {
      params.set("mode", nextMode);
    }
    if (nextUniversity) {
      params.set("university", nextUniversity);
    }
    if (nextSubjectTagID) {
      params.set("subject_tag_id", nextSubjectTagID);
    }

    const next = params.toString();
    router.replace(next ? `/search?${next}` : "/search");
  }, [router]);

  const fetchTags = useCallback(async () => {
    try {
      const response = await api.get<{ tags: RepositoryTag[] }>("/repositories/tags");
      setTags(response.data.tags || []);
    } catch {
      setTags([]);
    }
  }, []);

  const performContentSearch = useCallback(async (offset: number, append: boolean) => {
    const q = queryInput.trim();
    if (!q) {
      setHits([]);
      setUserHits([]);
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
        tag_id?: string;
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
      if (selectedSubjectTagID) {
        payload.tag_id = selectedSubjectTagID;
      }

      const response = await api.post<SearchResponse>("/search", payload);
      const nextHits = response.data.hits || [];
      setTotal(response.data.total || 0);
      setHits((prev) => (append ? [...prev, ...nextHits] : nextHits));
      if (!append) {
        setUserHits([]);
      }
    } catch (error) {
      toast.error(getErrorMessage(error, "Не удалось выполнить поиск"));
      if (!append) {
        setHits([]);
        setUserHits([]);
        setTotal(0);
      }
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [activeType, pageSize, queryInput, selectedSubjectTagID]);

  const performUserSearch = useCallback(async () => {
    const q = queryInput.trim();
    if (!q) {
      setUserHits([]);
      setHits([]);
      setTotal(0);
      return;
    }

    setLoading(true);
    try {
      const payload: {
        query: string;
        university?: string;
        tag_id?: string;
        limit: number;
        offset: number;
      } = {
        query: q,
        limit: 50,
        offset: 0,
      };

      if (selectedUniversity) {
        payload.university = selectedUniversity;
      }

      if (selectedSubjectTagID) {
        payload.tag_id = selectedSubjectTagID;
      }

      const response = await api.post<SearchUsersResponse>("/search/users", payload);

      setUserHits(response.data.users || []);
      setHits([]);
      setTotal(response.data.total || 0);
    } catch (error) {
      toast.error(getErrorMessage(error, "Не удалось выполнить поиск пользователей"));
      setUserHits([]);
      setHits([]);
      setTotal(0);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [queryInput, selectedSubjectTagID, selectedUniversity]);

  useEffect(() => {
    const urlQuery = searchParams.get("q") || "";
    const urlType = (searchParams.get("type") || "all") as SearchEntityType;
    const urlMode = (searchParams.get("mode") || "content") as SearchMode;
    const urlUniversity = searchParams.get("university") || "";
    const urlSubjectTagID = searchParams.get("subject_tag_id") || "";

    setQueryInput(urlQuery);
    setActiveType(urlType === "repository" || urlType === "document" || urlType === "file" ? urlType : "all");
    setActiveMode(urlMode === "users" ? "users" : "content");
    setSelectedUniversity(urlUniversity);
    setSelectedSubjectTagID(urlSubjectTagID);
  }, [searchParams]);

  useEffect(() => {
    void fetchTags();
  }, [fetchTags]);

  useEffect(() => {
    if (selectedSubjectTagID && !subjectOptions.some((item) => item.tag_id === selectedSubjectTagID)) {
      setSelectedSubjectTagID("");
    }
  }, [selectedSubjectTagID, subjectOptions]);

  useEffect(() => {
    if (!normalizedQuery) {
      setHits([]);
      setUserHits([]);
      setTotal(0);
      return;
    }
    if (activeMode === "users") {
      void performUserSearch();
      return;
    }
    void performContentSearch(0, false);
  }, [activeMode, activeType, normalizedQuery, performContentSearch, performUserSearch]);

  const handleSubmit = () => {
    updateUrl(queryInput, activeType, activeMode, selectedUniversity, selectedSubjectTagID);
  };

  const resolveRepository = useCallback(async (repoId: string): Promise<RepositoryResolve | null> => {
    const cached = repoCacheRef.current.get(repoId);
    if (cached) return cached;

    try {
      const response = await api.get(`/repositories/${repoId}`);
      const repo = (response.data.repository || response.data) as RepositoryResolve;
      const resolved = {
        owner_id: repo.owner_id,
        owner_username: repo.owner_username,
        slug: repo.slug,
      };
      repoCacheRef.current.set(repoId, resolved);
      setRepoLookupTick((value) => value + 1);
      return resolved;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const preloadRepositories = async () => {
      for (const hit of hits) {
        const repoId = hit.entity_type === "repository" ? hit.entity_id : hit.repo_id || "";
        if (!repoId || repoCacheRef.current.has(repoId)) {
          continue;
        }

        await resolveRepository(repoId);
        if (cancelled) {
          return;
        }
      }
    };

    void preloadRepositories();

    return () => {
      cancelled = true;
    };
  }, [hits, resolveRepository]);

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

    let owner = parsedRepoPath?.owner || repo?.owner_username || repo?.owner_id || hit.owner_id || "";
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
            {modes.map((mode) => (
              <Button
                key={mode}
                size="sm"
                variant={activeMode === mode ? "default" : "outline"}
                onClick={() => {
                  const nextType = mode === "users" ? "all" : activeType;
                  setActiveMode(mode);
                  updateUrl(queryInput, nextType, mode, selectedUniversity, selectedSubjectTagID);
                }}
              >
                {mode === "content" ? "Контент" : "Пользователи"}
              </Button>
            ))}
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div className="space-y-1">
              <Label>Вуз (опционально)</Label>
              <Select
                value={selectedUniversity || "all"}
                onValueChange={(value) => {
                  const nextUniversity = value === "all" ? "" : value;
                  setSelectedUniversity(nextUniversity);
                  setSelectedSubjectTagID("");
                  updateUrl(queryInput, activeType, activeMode, nextUniversity, "");
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Любой вуз" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Любой вуз</SelectItem>
                  {UNIVERSITY_OPTIONS.filter(Boolean).map((university) => (
                    <SelectItem key={university} value={university}>
                      {university}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1 md:col-span-2">
              <Label>Предмет (опционально)</Label>
              <Select
                value={selectedSubjectTagID || "all"}
                onValueChange={(value) => {
                  const nextSubjectTagID = value === "all" ? "" : value;
                  setSelectedSubjectTagID(nextSubjectTagID);
                  updateUrl(queryInput, activeType, activeMode, selectedUniversity, nextSubjectTagID);
                }}
                disabled={!selectedUniversity}
              >
                <SelectTrigger className="w-full">
                  <SelectValue
                    placeholder={selectedUniversity ? "Любой предмет" : "Сначала выберите вуз"}
                  >
                    {selectedSubject ? selectedSubject.name.replace(/^МИРЭА •\s*/, "").replace(/^МГУ •\s*/, "") : undefined}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Любой предмет</SelectItem>
                  {subjectOptions.map((subject) => (
                    <SelectItem key={subject.tag_id} value={subject.tag_id}>
                      {subject.name.replace(/^МИРЭА •\s*/, "").replace(/^МГУ •\s*/, "")}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            {activeMode === "content" ? typeFilters.map((type) => (
              <Button
                key={type}
                size="sm"
                variant={activeType === type ? "default" : "outline"}
                onClick={() => {
                  setActiveType(type);
                  updateUrl(queryInput, type, activeMode, selectedUniversity, selectedSubjectTagID);
                }}
              >
                {typeLabel(type)}
              </Button>
            )) : null}
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

          {activeMode === "users" && userHits.length > 0 ? (
            userHits.map((item) => (
              <Card key={item.user_id} className="hover:border-primary/40 transition-colors">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <Avatar size="sm">
                        <AvatarImage src={item.avatar_url || `https://github.com/identicons/${item.username}.png`} />
                        <AvatarFallback>{(item.username || "?").slice(0, 2).toUpperCase()}</AvatarFallback>
                      </Avatar>
                      <div className="min-w-0">
                        <CardTitle className="text-base truncate">{item.display_name || item.username}</CardTitle>
                        <CardDescription>@{item.username}</CardDescription>
                      </div>
                    </div>
                    {item.university ? (
                      <span className="shrink-0 rounded-full border px-2 py-1 text-xs font-normal text-muted-foreground">
                        {item.university}
                      </span>
                    ) : null}
                  </div>
                </CardHeader>
                <CardContent className="pt-0 flex items-center justify-between gap-4">
                  <div className="text-xs text-muted-foreground">
                    Совпадений по репозиториям: {item.repositories_count}
                  </div>
                  <Button size="sm" variant="outline" onClick={() => router.push(`/user/${item.username || item.user_id}`)}>
                    Открыть профиль <ArrowUpRight className="h-4 w-4 ml-1" />
                  </Button>
                </CardContent>
              </Card>
            ))
          ) : null}

          {activeMode === "content" && hits.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                Ничего не найдено.
              </CardContent>
            </Card>
          ) : null}

          {activeMode === "users" && userHits.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                Пользователи не найдены.
              </CardContent>
            </Card>
          ) : null}

          {activeMode === "content" && hits.length > 0 ? (
            hits.map((hit) => {
              const repoId = hit.entity_type === "repository" ? hit.entity_id : hit.repo_id || "";
              const resolvedRepo = repoId ? repoCacheRef.current.get(repoId) : null;

              return (
                <Card key={`${hit.entity_type}-${hit.entity_id}`} className="hover:border-primary/40 transition-colors">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base flex items-center justify-between gap-2">
                      <span className="flex items-center gap-2">
                        {iconByType(hit.entity_type)}
                        {formatHitTitle(hit, resolvedRepo)}
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
              );
            })
          ) : null}

          {hasMore ? (
            <div className="flex justify-center pt-2">
              <Button
                variant="outline"
                onClick={() => void performContentSearch(hits.length, true)}
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
