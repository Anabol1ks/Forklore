"use client";

import { Book, GitFork, Star, Settings, FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import Link from "next/link";
import { useRepo } from "./repo-context";
import { usePathname } from "next/navigation";
import { ReactNode } from "react";

export function RepoLayoutComponent({ owner, slug, children }: { owner: string; slug: string; children: ReactNode }) {
  const { repo, parentRepo, loading, errorState, starsCount, isStarred, isStarLoading, handleToggleStar } = useRepo();
  const pathname = usePathname();

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

  // Active tab check
  const isSettings = pathname.includes("/settings");

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2 text-2xl">
            <Book className="h-6 w-6 text-muted-foreground" />
            <Link href={`/${owner}`} className="text-primary hover:underline cursor-pointer">
              {owner}
            </Link>
            <span className="text-muted-foreground">/</span>
            <Link href={`/${owner}/${slug}`} className="font-bold hover:underline">
              {repo.name}
            </Link>
            <span className="ml-2 px-2 py-0.5 text-xs border rounded-full text-muted-foreground">
              {repo.visibility}
            </span>
          </div>
          {parentRepo && (
            <div className="text-sm text-muted-foreground ml-8 flex items-center gap-1">
              <span>forked from</span>
              <Link
                href={`/${parentRepo.owner_username || parentRepo.owner_id || "unknown"}/${parentRepo.slug}`}
                className="hover:text-primary hover:underline"
              >
                {parentRepo.owner_username || parentRepo.owner_id || "unknown"}/{parentRepo.slug}
              </Link>
            </div>
          )}
        </div>

        <div className="flex gap-2 h-8">
          <Button variant="outline" size="sm" onClick={handleToggleStar} disabled={isStarLoading} className="h-8">
            <Star className={`mr-2 h-4 w-4 ${isStarred ? "fill-current text-yellow-500" : ""}`} /> 
            {isStarred ? "Starred" : "Star"} 
            <span className="ml-2 text-muted-foreground bg-muted px-1.5 py-0.5 rounded-full text-xs">{starsCount}</span>
          </Button>
          <Button variant="outline" size="sm" className="h-8">
            <GitFork className="mr-2 h-4 w-4" /> Fork 
            <span className="ml-2 text-muted-foreground bg-muted px-1.5 py-0.5 rounded-full text-xs">0</span>
          </Button>
        </div>
      </div>

      <p className="text-lg text-muted-foreground">{repo.description || ""}</p>

      {/* GitHub-like Tabs Navigation */}
      <div className="border-b">
        <nav className="flex gap-1">
          <Link
            href={`/${owner}/${slug}`}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              !isSettings
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground hover:border-muted"
            }`}
          >
            <FileText className="h-4 w-4" /> Code
          </Link>
          <Link
            href={`/${owner}/${slug}/settings`}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              isSettings
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground hover:border-muted"
            }`}
          >
            <Settings className="h-4 w-4" /> Settings
          </Link>
        </nav>
      </div>

      <div className="pt-2">
        {children}
      </div>
    </div>
  );
}
