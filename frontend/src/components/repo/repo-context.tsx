"use client";

import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from "react";
import { api } from "@/lib/api";
import axios from "axios";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";

export interface Repository {
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

export function getId(obj: Record<string, unknown> | null | undefined, keys: string[]): string {
  if (!obj) return "";
  for (const key of keys) {
    const value = obj[key];
    if (typeof value === "string" && value.length > 0) {
      return value;
    }
  }
  return "";
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (axios.isAxiosError(error)) {
    return (error.response?.data as { message?: string } | undefined)?.message || fallback;
  }
  return fallback;
}

interface RepoContextState {
  repo: Repository | null;
  parentRepo: Repository | null;
  loading: boolean;
  errorState: "not-found" | "forbidden" | null;
  starsCount: number;
  isStarred: boolean;
  isOwner: boolean;
  repoId: string;
  isStarLoading: boolean;
  fetchRepo: () => Promise<void>;
  handleToggleStar: () => Promise<void>;
}

const RepoContext = createContext<RepoContextState | undefined>(undefined);

export function RepoProvider({ owner, slug, children }: { owner: string; slug: string; children: ReactNode }) {
  const [repo, setRepo] = useState<Repository | null>(null);
  const [parentRepo, setParentRepo] = useState<Repository | null>(null);
  const [loading, setLoading] = useState(true);
  const [errorState, setErrorState] = useState<"not-found" | "forbidden" | null>(null);
  const [starsCount, setStarsCount] = useState(0);
  const [isStarred, setIsStarred] = useState(false);
  const [isStarLoading, setStarLoading] = useState(false);

  const [repoId, setRepoId] = useState<string>("");
  const router = useRouter();
  const { user } = useAuthStore();
  const isOwner = !!user?.id && !!repo?.owner_id && user.id === repo.owner_id;

  const fetchRepo = useCallback(async () => {
    try {
      setLoading(true);
      setErrorState(null);
      const response = await api.get(`/users/${owner}/repositories/${slug}`);
      const repoData = (response.data.repository || response.data) as Repository;
      const parsedRepoId = getId(repoData as unknown as Record<string, unknown>, ["id", "repo_id"]);

      if (!parsedRepoId) {
        throw new Error("Repository id is missing in response");
      }
      setRepoId(parsedRepoId);

      const [repoByIdResponse] = await Promise.all([
        api.get(`/repositories/${parsedRepoId}`),
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

      try {
        const starRes = await api.get(`/repositories/${parsedRepoId}/star`);
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
    void fetchRepo();
  }, [fetchRepo]);

  const handleToggleStar = async () => {
    if (!repoId || isStarLoading) return;

    try {
      setStarLoading(true);
      const res = await api.post(`/repositories/${repoId}/star`);
      setIsStarred(Boolean(res.data.starred));
      setStarsCount(Number(res.data.stars_count || 0));
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        toast.error("Для Star нужно войти в аккаунт");
        router.push("/login");
      } else {
        toast.error(getErrorMessage(error, "Не удалось обновить star"));
      }
    } finally {
      setStarLoading(false);
    }
  };

  return (
    <RepoContext.Provider
      value={{
        repo,
        parentRepo,
        loading,
        errorState,
        starsCount,
        isStarred,
        isOwner,
        repoId,
        isStarLoading,
        fetchRepo,
        handleToggleStar,
      }}
    >
      {children}
    </RepoContext.Provider>
  );
}

export function useRepo() {
  const context = useContext(RepoContext);
  if (!context) {
    throw new Error("useRepo must be used within a RepoProvider");
  }
  return context;
}
