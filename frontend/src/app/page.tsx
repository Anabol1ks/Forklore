"use client";

import { useAuthStore } from "@/store/auth";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { GitFork, BookOpen, Clock, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useRouter } from "next/navigation";

interface Repository {
  id: string;
  name: string;
  slug: string;
  description: string;
  visibility: string;
  updated_at: string;
  stats?: {
    forks?: number;
  };
}

export default function Home() {
  const { user, isAuthenticated } = useAuthStore();
  const router = useRouter();
  const [repos, setRepos] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(isAuthenticated);

  useEffect(() => {
    if (!isAuthenticated) {
      setRepos([]);
      setLoading(false);
      return;
    }

    let isMounted = true;
    const loadRepos = async () => {
      setLoading(true);
      try {
        const res = await api.get("/repositories/me");
        if (isMounted) {
          setRepos(res.data.repositories || []);
        }
      } catch (err) {
        console.error("Failed to load repos", err);
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    void loadRepos();

    return () => {
      isMounted = false;
    };
  }, [isAuthenticated]);

  return (
    <div className="space-y-8">
      {!isAuthenticated ? (
        <div className="flex flex-col items-center justify-center space-y-6 text-center py-20">
          <BookOpen className="h-16 w-16 text-muted-foreground" />
          <h1 className="text-4xl font-bold tracking-tighter sm:text-5xl md:text-6xl">
            Учебные материалы по-новому
          </h1>
          <p className="max-w-150 text-lg text-muted-foreground">
            Forklore — это платформа для хранения, ведения и распространения учебных материалов.
            Создавайте базы знаний, развивайте их вместе с сообществом и делайте форки.
          </p>
          <div className="flex gap-4">
            <Link href="/register">
              <Button size="lg">Начать использование</Button>
            </Link>
            <Link href="/login">
              <Button size="lg" variant="outline">Уже есть аккаунт?</Button>
            </Link>
          </div>
        </div>
      ) : (
        <div className="space-y-6">
          <div className="flex justify-between items-center">
            <h2 className="text-3xl font-bold tracking-tight">Ваши репозитории</h2>
            <Link href="/repo/create">
              <Button>Создать репозиторий</Button>
            </Link>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {loading ? (
              <div className="col-span-full flex justify-center py-10">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : (
              repos.map((repo) => (
                <Card key={repo.id} className="hover:border-primary/50 transition-colors">
                  <CardHeader>
                    <div className="flex justify-between items-start">
                      <div className="flex items-center gap-2">
                        <BookOpen className="h-5 w-5 text-primary" />
                        <Link href={`/${user?.username}/${repo.slug}`} className="hover:underline">
                          <CardTitle className="text-xl">{repo.name}</CardTitle>
                        </Link>
                      </div>
                      <span className="text-xs text-muted-foreground border px-2 py-1 rounded-full capitalize">
                        {repo.visibility}
                      </span>
                    </div>
                    <CardDescription>
                      {repo.description || "Без описания"}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex gap-4 text-sm text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <GitFork className="h-4 w-4" /> {repo.stats?.forks || 0}
                      </span>
                      <span className="flex items-center gap-1">
                        <Clock className="h-4 w-4" /> 
                        {repo.updated_at ? new Date(repo.updated_at).toLocaleDateString("ru-RU") : "Н/Д"}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))
            )}

            <Card className="flex flex-col items-center justify-center p-6 border-dashed text-muted-foreground hover:text-foreground hover:border-foreground transition-colors cursor-pointer" onClick={() => router.push('/repo/create')}>
              <div className="h-10 w-10 bg-muted rounded-full flex items-center justify-center mb-2">
                <span className="text-xl">+</span>
              </div>
              <span className="font-medium">Создать новый репозиторий</span>
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
