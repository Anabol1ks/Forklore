"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Book, Users, Star } from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuthStore } from "@/store/auth";

interface UserProfile {
  id: string;
  username: string;
  bio: string;
  stats: {
    followers: number;
    following: number;
  };
}

interface UserRepo {
  id?: string;
  repo_id?: string;
  owner_username?: string;
  name: string;
  slug?: string;
  visibility?: string;
  description?: string;
}

export default function UserProfilePage() {
  const params = useParams<{ ownerId: string }>();
  const ownerId = params.ownerId;
  const { user } = useAuthStore();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [repos, setRepos] = useState<UserRepo[]>([]);
  const [starredRepos, setStarredRepos] = useState<UserRepo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchUserRepos = async () => {
      try {
        setLoading(true);
        // Заглушка профиля (нет отдельного эндпоинта для юзера, используем ownerId как имя)
        setProfile({
          id: ownerId,
          username: ownerId,
          bio: "Пользователь платформы",
          stats: { followers: 0, following: 0 }
        });
        
        // Получаем репозитории
        const response = await api.get(`/users/${ownerId}/repositories`);
        setRepos(response.data.repositories || []);

        const isSelf = !!user && (user.username === ownerId || user.id === ownerId);
        if (isSelf) {
          try {
            const starsResponse = await api.get(`/repositories/me/starred`);
            setStarredRepos(starsResponse.data.repositories || []);
          } catch {
            setStarredRepos([]);
          }
        } else {
          setStarredRepos([]);
        }
        
        setLoading(false);
      } catch (error) {
        console.error("Failed to load user repositories", error);
        setLoading(false);
      }
    }
    
    if (ownerId) {
      fetchUserRepos();
    }
  }, [ownerId, user]);

  if (loading) {
    return <div className="space-y-4 animate-pulse"><Skeleton className="h-32 w-32 rounded-full" /></div>;
  }

  if (!profile) return <div>Пользователь не найден</div>;

  return (
    <div className="flex flex-col md:flex-row gap-8">
      {/* Sidebar - Profile Info */}
      <div className="w-full md:w-1/4 space-y-4">
        <Avatar className="w-full h-auto aspect-square">
          <AvatarImage src={`https://github.com/identicons/${profile.username}.png`} />
          <AvatarFallback>{profile.username.substring(0,2).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div>
          <h1 className="text-2xl font-bold">{profile.username}</h1>
          <p className="text-muted-foreground">{profile.bio}</p>
        </div>
        <Button className="w-full" variant="outline">Подписаться</Button>
        <div className="flex gap-4 text-sm text-muted-foreground">
          <span className="flex items-center gap-1 hover:text-primary cursor-pointer">
            <Users className="h-4 w-4" /> {profile.stats.followers} followers
          </span>
          <span className="hover:text-primary cursor-pointer">
            {profile.stats.following} following
          </span>
        </div>
        <div className="text-xs text-muted-foreground opacity-50 mt-4">
          [Blocked] Нет API для полноценного профиля (followers, bio).
        </div>
      </div>

      {/* Main Content */}
      <div className="w-full md:w-3/4">
        <Tabs defaultValue="repositories" className="w-full">
          <TabsList className="mb-4">
            <TabsTrigger value="repositories" className="flex items-center gap-2">
              <Book className="h-4 w-4" /> Репозитории
              <span className="ml-1 rounded-full bg-muted px-2 py-0.5 text-xs">{repos.length}</span>
            </TabsTrigger>
            <TabsTrigger value="stars" className="flex items-center gap-2">
              <Star className="h-4 w-4" /> Избранное
            </TabsTrigger>
          </TabsList>

          <TabsContent value="repositories" className="space-y-4">
            <Input placeholder="Найти репозиторий..." className="max-w-md" />
            <div className="grid grid-cols-1 gap-4">
              {repos.map((repo) => (
                <Card key={repo.repo_id || repo.id || repo.name} className="pt-2 hover:border-primary/50 transition">
                  <CardHeader className="py-4">
                    <div className="flex justify-between">
                      <Link href={`/${profile.username}/${repo.slug || repo.name}`} className="hover:underline">
                        <CardTitle className="text-xl text-primary flex items-center gap-2">
                          {repo.name} <span className="text-xs border rounded-full px-2 py-1 text-muted-foreground bg-background">{repo.visibility || "public"}</span>
                        </CardTitle>
                      </Link>
                    </div>
                    <CardDescription>{repo.description || "Без описания"}</CardDescription>
                  </CardHeader>
                </Card>
              ))}
            </div>
          </TabsContent>

          <TabsContent value="stars">
            <div className="grid grid-cols-1 gap-4">
              {starredRepos.length > 0 ? starredRepos.map((repo) => (
                <Card key={repo.repo_id || repo.id || repo.name} className="pt-2 hover:border-primary/50 transition">
                  <CardHeader className="py-4">
                    <div className="flex justify-between">
                      <Link href={`/${repo.owner_username || profile.username}/${repo.slug || repo.name}`} className="hover:underline">
                        <CardTitle className="text-xl text-primary flex items-center gap-2">
                          {repo.name} <span className="text-xs border rounded-full px-2 py-1 text-muted-foreground bg-background">{repo.visibility || "public"}</span>
                        </CardTitle>
                      </Link>
                    </div>
                    <CardDescription>{repo.description || "Без описания"}</CardDescription>
                  </CardHeader>
                </Card>
              )) : (
                <div className="p-10 text-center border rounded-md text-muted-foreground">
                  Пользователь пока ничего не добавил в избранное.
                </div>
              )}
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
