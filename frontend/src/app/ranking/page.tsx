"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import axios from "axios";
import { Trophy, Flame, BookMarked, Users, Star, GitFork, Activity, Medal, ChevronRight } from "lucide-react";

import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";

type RankingMode = "overall" | "monthly" | "subject";

type RepositoryTag = {
  tag_id: string;
  name: string;
};

type RankingEntry = {
  user_id: string;
  tag_id?: string;
  username?: string;
  display_name?: string;
  avatar_url?: string;
  title_label?: string;
  score: number;
  followers_count: number;
  followers_gained_30d: number;
  stars_received_total: number;
  stars_received_30d: number;
  forks_received_total: number;
  forks_received_30d: number;
  public_repositories_count: number;
  activity_points_total: number;
  activity_points_30d: number;
  active_weeks_last_8: number;
  active_months_count: number;
  subject_score: number;
};

type RankingResponse = {
  entries: RankingEntry[];
  total: number;
};

const PAGE_SIZE = 20;
const UNIVERSITY_OPTIONS = ["", "МИРЭА", "МГУ"] as const;

function getErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error)) {
    return (error.response?.data as { message?: string } | undefined)?.message || fallback;
  }
  return fallback;
}

function normalizeUniversity(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized.includes("мирэа")) {
    return "МИРЭА";
  }
  if (normalized.includes("мгу")) {
    return "МГУ";
  }
  return "";
}

function metricLabel(mode: RankingMode): string {
  switch (mode) {
    case "monthly":
      return "Рейтинг за 30 дней";
    case "subject":
      return "Рейтинг по предмету";
    default:
      return "Общий рейтинг";
  }
}

function scoreForMode(entry: RankingEntry, mode: RankingMode): number {
  if (mode === "subject") {
    return entry.subject_score || entry.score;
  }
  return entry.score;
}

function avatarFallback(entry: RankingEntry): string {
  const source = entry.display_name || entry.username || "U";
  return source.slice(0, 2).toUpperCase();
}

export default function RankingPage() {
  const [mode, setMode] = useState<RankingMode>("overall");
  const [entries, setEntries] = useState<RankingEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);

  const [tags, setTags] = useState<RepositoryTag[]>([]);
  const [selectedUniversity, setSelectedUniversity] = useState<string>("");
  const [selectedSubjectTagID, setSelectedSubjectTagID] = useState<string>("");

  const selectedSubject = useMemo(
    () => tags.find((item) => item.tag_id === selectedSubjectTagID),
    [selectedSubjectTagID, tags]
  );

  const subjectOptions = useMemo(() => {
    if (!selectedUniversity) {
      return [] as RepositoryTag[];
    }

    return tags.filter((tag) => normalizeUniversity(tag.name) === selectedUniversity);
  }, [selectedUniversity, tags]);

  const topThree = useMemo(() => entries.slice(0, 3), [entries]);
  const hasMore = entries.length < total;

  const fetchTags = useCallback(async () => {
    try {
      const response = await api.get<{ tags: RepositoryTag[] }>("/repositories/tags");
      setTags(response.data.tags || []);
    } catch {
      setTags([]);
    }
  }, []);

  const fetchLeaderboard = useCallback(
    async (append: boolean, offsetOverride?: number) => {
      if (mode === "subject" && !selectedSubjectTagID) {
        setEntries([]);
        setTotal(0);
        setLoading(false);
        setLoadingMore(false);
        return;
      }

      if (append) {
        setLoadingMore(true);
      } else {
        setLoading(true);
      }

      try {
        const offset = append ? offsetOverride ?? 0 : 0;
        let path = `/rankings/${mode}`;
        if (mode === "subject") {
          path = `/rankings/subject/${selectedSubjectTagID}`;
        }

        const response = await api.get<RankingResponse>(path, {
          params: { limit: PAGE_SIZE, offset },
        });

        const nextEntries = response.data.entries || [];
        setTotal(response.data.total || 0);
        setEntries((prev) => (append ? [...prev, ...nextEntries] : nextEntries));
      } catch (error) {
        toast.error(getErrorMessage(error, "Не удалось загрузить рейтинг"));
        if (!append) {
          setEntries([]);
          setTotal(0);
        }
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [mode, selectedSubjectTagID]
  );

  useEffect(() => {
    void fetchTags();
  }, [fetchTags]);

  useEffect(() => {
    if (selectedSubjectTagID && !subjectOptions.some((item) => item.tag_id === selectedSubjectTagID)) {
      setSelectedSubjectTagID("");
    }
  }, [selectedSubjectTagID, subjectOptions]);

  useEffect(() => {
    setEntries([]);
    setTotal(0);
    void fetchLeaderboard(false);
  }, [mode, selectedSubjectTagID, fetchLeaderboard]);

  return (
    <div className="space-y-6">
      <section className="relative overflow-hidden rounded-2xl border bg-gradient-to-br from-primary/15 via-background to-emerald-500/10 p-6 md:p-8">
        <div className="absolute -right-16 -top-16 h-40 w-40 rounded-full bg-primary/15 blur-2xl" />
        <div className="absolute -left-20 bottom-0 h-36 w-36 rounded-full bg-emerald-500/15 blur-2xl" />

        <div className="relative z-10 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div className="space-y-2">
            <p className="inline-flex items-center gap-2 rounded-full border bg-background/75 px-3 py-1 text-xs font-medium text-muted-foreground">
              <Trophy className="h-3.5 w-3.5" />
              Таблица лидеров сообщества
            </p>
            <h1 className="text-3xl font-bold tracking-tight">Рейтинг Forklore</h1>
            <p className="max-w-2xl text-sm text-muted-foreground">
              Сравнивайте вклад участников по общей активности, динамике за месяц и предметным направлениям.
            </p>
          </div>

          <Tabs value={mode} onValueChange={(value) => setMode(value as RankingMode)} className="w-full md:w-auto">
            <TabsList>
              <TabsTrigger value="overall" className="gap-2"><Trophy className="h-4 w-4" />Общий</TabsTrigger>
              <TabsTrigger value="monthly" className="gap-2"><Flame className="h-4 w-4" />Месяц</TabsTrigger>
              <TabsTrigger value="subject" className="gap-2"><BookMarked className="h-4 w-4" />Предмет</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </section>

      {mode === "subject" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Фильтр предметного рейтинга</CardTitle>
            <CardDescription>Выберите вуз и предметный тег, чтобы построить лидерборд по конкретному направлению.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="ranking-university">Вуз</Label>
              <Select value={selectedUniversity || "all"} onValueChange={(value) => {
                const nextValue = value ?? "all";
                setSelectedUniversity(nextValue === "all" ? "" : nextValue);
              }}>
                <SelectTrigger id="ranking-university">
                  <SelectValue placeholder="Любой вуз" />
                </SelectTrigger>
                <SelectContent>
                  {UNIVERSITY_OPTIONS.map((option) => (
                    <SelectItem key={option || "all"} value={option || "all"}>
                      {option || "Любой вуз"}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="ranking-subject">Предмет</Label>
              <Select
                value={selectedSubjectTagID || "none"}
                onValueChange={(value) => {
                  const nextValue = value ?? "none";
                  setSelectedSubjectTagID(nextValue === "none" ? "" : nextValue);
                }}
                disabled={!selectedUniversity}
              >
                <SelectTrigger id="ranking-subject">
                  <SelectValue placeholder={selectedUniversity ? "Выберите предмет" : "Сначала выберите вуз"} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Не выбрано</SelectItem>
                  {subjectOptions.map((tag) => (
                    <SelectItem key={tag.tag_id} value={tag.tag_id}>{tag.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-3">
        {loading
          ? Array.from({ length: 3 }).map((_, index) => (
              <Card key={`skeleton-top-${index}`}>
                <CardHeader><Skeleton className="h-5 w-20" /></CardHeader>
                <CardContent className="space-y-3">
                  <Skeleton className="h-10 w-10 rounded-full" />
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-4 w-1/2" />
                </CardContent>
              </Card>
            ))
          : topThree.map((entry, index) => {
              const score = scoreForMode(entry, mode);
              const username = entry.username || entry.user_id;
              const profileLink = `/user/${encodeURIComponent(username)}`;

              return (
                <Card key={`${entry.user_id}-${index}`} className={cn(index === 0 && "border-primary/50") }>
                  <CardHeader>
                    <CardTitle className="text-base inline-flex items-center gap-2">
                      <Medal className={cn("h-4 w-4", index === 0 && "text-yellow-500", index === 1 && "text-slate-400", index === 2 && "text-amber-700")} />
                      #{index + 1}
                    </CardTitle>
                    <CardDescription>{metricLabel(mode)}</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex items-center gap-3">
                      <Avatar className="h-11 w-11 border">
                        <AvatarImage src={entry.avatar_url} alt={username} />
                        <AvatarFallback>{avatarFallback(entry)}</AvatarFallback>
                      </Avatar>
                      <div className="min-w-0">
                        <Link href={profileLink} className="font-medium hover:underline">
                          {entry.display_name || username}
                        </Link>
                        <p className="truncate text-xs text-muted-foreground">@{username}</p>
                      </div>
                    </div>
                    <div className="rounded-lg border bg-muted/30 px-3 py-2">
                      <p className="text-xs text-muted-foreground">Скор</p>
                      <p className="text-2xl font-semibold tracking-tight">{score.toFixed(2)}</p>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Участники</CardTitle>
          <CardDescription>
            {mode === "subject" && selectedSubject ? `Тег: ${selectedSubject.name}` : metricLabel(mode)}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {loading ? (
            Array.from({ length: 6 }).map((_, index) => <Skeleton key={`row-skeleton-${index}`} className="h-16 w-full" />)
          ) : entries.length === 0 ? (
            <div className="rounded-lg border border-dashed p-8 text-center text-muted-foreground">
              {mode === "subject" && !selectedSubjectTagID
                ? "Выберите предметный тег для отображения рейтинга."
                : "Пока нет данных для отображения рейтинга."}
            </div>
          ) : (
            entries.map((entry, index) => {
              const rank = index + 1;
              const score = scoreForMode(entry, mode);
              const username = entry.username || entry.user_id;
              const profileLink = `/user/${encodeURIComponent(username)}`;

              return (
                <div key={`${entry.user_id}-${rank}`} className="rounded-xl border bg-card/70 p-3 md:p-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="mt-0.5 w-8 text-sm font-semibold text-muted-foreground">#{rank}</div>
                      <Avatar className="h-10 w-10 border">
                        <AvatarImage src={entry.avatar_url} alt={username} />
                        <AvatarFallback>{avatarFallback(entry)}</AvatarFallback>
                      </Avatar>
                      <div className="min-w-0">
                        <Link href={profileLink} className="font-medium hover:underline">
                          {entry.display_name || username}
                        </Link>
                        <p className="truncate text-xs text-muted-foreground">@{username}</p>
                        {entry.title_label ? <p className="text-xs text-muted-foreground">{entry.title_label}</p> : null}
                      </div>
                    </div>

                    <div className="text-right">
                      <p className="text-xs text-muted-foreground">Скор</p>
                      <p className="text-lg font-semibold">{score.toFixed(2)}</p>
                    </div>
                  </div>

                  <div className="mt-3 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2 lg:grid-cols-4">
                    <div className="inline-flex items-center gap-1.5"><Users className="h-3.5 w-3.5" />Подписчики: {entry.followers_count}</div>
                    <div className="inline-flex items-center gap-1.5"><Star className="h-3.5 w-3.5" />Звезды: {mode === "monthly" ? entry.stars_received_30d : entry.stars_received_total}</div>
                    <div className="inline-flex items-center gap-1.5"><GitFork className="h-3.5 w-3.5" />Форки: {mode === "monthly" ? entry.forks_received_30d : entry.forks_received_total}</div>
                    <div className="inline-flex items-center gap-1.5"><Activity className="h-3.5 w-3.5" />Активность: {mode === "monthly" || mode === "subject" ? entry.activity_points_30d : entry.activity_points_total}</div>
                  </div>
                </div>
              );
            })
          )}

          {hasMore && !loading && (
            <div className="flex justify-center pt-2">
              <Button onClick={() => void fetchLeaderboard(true, entries.length)} disabled={loadingMore} variant="outline" className="gap-2">
                {loadingMore ? "Загрузка..." : "Показать ещё"}
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
