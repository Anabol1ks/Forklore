"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Book, Users, Star } from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuthStore } from "@/store/auth";

type ProfileTitle = {
  code: string;
  label: string;
};

type SocialLink = {
  social_link_id: string;
  platform: string;
  url: string;
  label?: string;
  position: number;
  is_visible: boolean;
};

interface UserProfile {
  user_id: string;
  username: string;
  display_name: string;
  bio?: string;
  avatar_url?: string;
  cover_url?: string;
  location?: string;
  website_url?: string;
  readme_markdown?: string;
  is_public: boolean;
  title?: ProfileTitle;
  followers_count: number;
  following_count: number;
  social_links: SocialLink[];
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

interface ListResponse<T> {
  profiles: T[];
  total: number;
}

interface ProfilePreview {
  user_id: string;
  username: string;
}

type ProfileUpdateForm = {
  display_name: string;
  bio: string;
  avatar_url: string;
  cover_url: string;
  location: string;
  website_url: string;
  is_public: boolean;
};

type SocialLinkForm = {
  social_link_id: string;
  platform: string;
  url: string;
  label: string;
  position: number;
  is_visible: boolean;
};

const SOCIAL_PLATFORMS = ["telegram", "github", "vk", "linkedin", "x", "youtube", "website", "other"];

function normalizeSocialURLInput(platform: string, rawURL: string): string | null {
  let value = rawURL.trim();
  if (!value) {
    return null;
  }

  if (value.startsWith("@")) {
    const handle = value.slice(1).trim();
    if (!handle) {
      return null;
    }

    switch (platform) {
      case "telegram":
        value = `https://t.me/${handle}`;
        break;
      case "github":
        value = `https://github.com/${handle}`;
        break;
      case "vk":
        value = `https://vk.com/${handle}`;
        break;
      case "linkedin":
        value = `https://www.linkedin.com/in/${handle}`;
        break;
      case "x":
        value = `https://x.com/${handle}`;
        break;
      default:
        return null;
    }
  }

  if (!/^https?:\/\//i.test(value)) {
    value = `https://${value}`;
  }

  try {
    const parsed = new URL(value);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return null;
    }
    if (!parsed.hostname) {
      return null;
    }
    return parsed.toString();
  } catch {
    return null;
  }
}

function isUUID(value: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value);
}

export default function UserProfilePage() {
  const params = useParams<{ ownerId: string }>();
  const ownerId = params.ownerId;
  const { user } = useAuthStore();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [repos, setRepos] = useState<UserRepo[]>([]);
  const [starredRepos, setStarredRepos] = useState<UserRepo[]>([]);
  const [titles, setTitles] = useState<ProfileTitle[]>([]);
  const [isFollowing, setIsFollowing] = useState(false);
  const [profileForm, setProfileForm] = useState<ProfileUpdateForm>({
    display_name: "",
    bio: "",
    avatar_url: "",
    cover_url: "",
    location: "",
    website_url: "",
    is_public: true,
  });
  const [readmeMarkdown, setReadmeMarkdown] = useState("");
  const [titleCode, setTitleCode] = useState("");
  const [socialForm, setSocialForm] = useState<SocialLinkForm>({
    social_link_id: "",
    platform: "github",
    url: "",
    label: "",
    position: 0,
    is_visible: true,
  });
  const [notice, setNotice] = useState<string | null>(null);
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingReadme, setSavingReadme] = useState(false);
  const [savingTitle, setSavingTitle] = useState(false);
  const [savingSocial, setSavingSocial] = useState(false);
  const [togglingFollow, setTogglingFollow] = useState(false);
  const [loading, setLoading] = useState(true);

  const isSelf = useMemo(() => {
    if (!user || !profile) {
      return false;
    }
    return user.id === profile.user_id || user.username === profile.username;
  }, [profile, user]);

  const displayName = profile?.display_name?.trim() || profile?.username || ownerId;

  const applyProfileToForms = (nextProfile: UserProfile) => {
    setProfileForm({
      display_name: nextProfile.display_name || nextProfile.username,
      bio: nextProfile.bio || "",
      avatar_url: nextProfile.avatar_url || "",
      cover_url: nextProfile.cover_url || "",
      location: nextProfile.location || "",
      website_url: nextProfile.website_url || "",
      is_public: nextProfile.is_public,
    });
    setReadmeMarkdown(nextProfile.readme_markdown || "");
    setTitleCode(nextProfile.title?.code || "");
  };

  const loadProfile = useCallback(async (owner: string): Promise<UserProfile> => {
    const ownerKey = owner.trim();
    const ownerLower = ownerKey.toLowerCase();

    const isSelfByRoute = !!user && (
      user.id === ownerKey ||
      user.username === ownerKey ||
      user.username.toLowerCase() === ownerLower
    );

    if (isSelfByRoute) {
      const response = await api.get("/profiles/me");
      return response.data.profile as UserProfile;
    }

    if (isUUID(ownerKey)) {
      const response = await api.get(`/profiles/by-user/${ownerKey}`);
      return response.data.profile as UserProfile;
    }

    const response = await api.get(`/profiles/by-username/${ownerLower}`);
    return response.data.profile as UserProfile;
  }, [user]);

  const refreshMyProfile = async () => {
    const response = await api.get("/profiles/me");
    const nextProfile = response.data.profile as UserProfile;
    setProfile(nextProfile);
    applyProfileToForms(nextProfile);
  };

  const loadFollowState = useCallback(async (targetProfile: UserProfile) => {
    if (!user) {
      setIsFollowing(false);
      return;
    }

    if (user.id === targetProfile.user_id || user.username === targetProfile.username) {
      setIsFollowing(false);
      return;
    }

    try {
      const response = await api.get<ListResponse<ProfilePreview>>(`/profiles/${targetProfile.user_id}/followers`, {
        params: { limit: 100, offset: 0 },
      });

      const exists = (response.data.profiles || []).some(
        (item) => item.user_id === user.id || item.username === user.username,
      );
      setIsFollowing(exists);
    } catch {
      setIsFollowing(false);
    }
  }, [user]);

  const loadTitles = useCallback(async () => {
    try {
      const response = await api.get("/profiles/titles");
      setTitles(response.data.titles || []);
    } catch {
      setTitles([]);
    }
  }, []);

  useEffect(() => {
    const fetchUserData = async () => {
      try {
        setLoading(true);
        setNotice(null);

        const profileResult = await loadProfile(ownerId);

        let reposResult;
        try {
          reposResult = await api.get(`/users/${profileResult.username}/repositories`);
        } catch {
          reposResult = await api.get(`/users/${ownerId}/repositories`);
        }

        setProfile(profileResult);
        applyProfileToForms(profileResult);
        setRepos(reposResult.data.repositories || []);

        const ownsPage = !!user && (user.username === profileResult.username || user.id === profileResult.user_id);

        if (ownsPage) {
          try {
            const starsResponse = await api.get(`/repositories/me/starred`);
            setStarredRepos(starsResponse.data.repositories || []);
          } catch {
            setStarredRepos([]);
          }

          await loadTitles();
        } else {
          setStarredRepos([]);
          setTitles([]);
        }

        await loadFollowState(profileResult);
        setLoading(false);
      } catch (error) {
        console.error("Failed to load user profile", error);
        setNotice("Не удалось загрузить профиль пользователя.");
        setLoading(false);
      }
    };

    if (ownerId) {
      void fetchUserData();
    }
  }, [loadFollowState, loadProfile, loadTitles, ownerId, user]);

  const handleFollowToggle = async () => {
    if (!profile || !user || isSelf) {
      return;
    }

    setTogglingFollow(true);
    setNotice(null);
    try {
      if (isFollowing) {
        await api.delete(`/profiles/${profile.user_id}/follow`);
        setIsFollowing(false);
        setProfile((prev) => (prev ? { ...prev, followers_count: Math.max(0, prev.followers_count - 1) } : prev));
      } else {
        await api.post(`/profiles/${profile.user_id}/follow`);
        setIsFollowing(true);
        setProfile((prev) => (prev ? { ...prev, followers_count: prev.followers_count + 1 } : prev));
      }
    } catch {
      setNotice("Не удалось обновить подписку. Попробуйте снова.");
    } finally {
      setTogglingFollow(false);
    }
  };

  const handleSaveProfile = async () => {
    setSavingProfile(true);
    setNotice(null);
    try {
      const response = await api.patch("/profiles/me", profileForm);
      const nextProfile = response.data.profile as UserProfile;
      setProfile(nextProfile);
      applyProfileToForms(nextProfile);
      setNotice("Профиль обновлен.");
    } catch {
      setNotice("Не удалось обновить профиль.");
    } finally {
      setSavingProfile(false);
    }
  };

  const handleSaveReadme = async () => {
    setSavingReadme(true);
    setNotice(null);
    try {
      const response = await api.patch("/profiles/me/readme", { readme_markdown: readmeMarkdown });
      const nextProfile = response.data.profile as UserProfile;
      setProfile(nextProfile);
      applyProfileToForms(nextProfile);
      setNotice("README профиля обновлен.");
    } catch {
      setNotice("Не удалось обновить README.");
    } finally {
      setSavingReadme(false);
    }
  };

  const handleSaveTitle = async () => {
    if (!titleCode.trim()) {
      setNotice("Выберите титул.");
      return;
    }

    setSavingTitle(true);
    setNotice(null);
    try {
      const response = await api.put("/profiles/me/title", { title_code: titleCode });
      const nextProfile = response.data.profile as UserProfile;
      setProfile(nextProfile);
      applyProfileToForms(nextProfile);
      setNotice("Титул профиля обновлен.");
    } catch {
      setNotice("Не удалось обновить титул.");
    } finally {
      setSavingTitle(false);
    }
  };

  const handleSubmitSocialLink = async () => {
    if (!socialForm.platform.trim() || !socialForm.url.trim()) {
      setNotice("Укажите platform и url.");
      return;
    }

    const normalizedSocialURL = normalizeSocialURLInput(socialForm.platform, socialForm.url);
    if (!normalizedSocialURL) {
      setNotice("Укажите корректный URL социальной ссылки (например: https://github.com/username).");
      return;
    }

    setSavingSocial(true);
    setNotice(null);
    try {
      const payload = {
        social_link_id: socialForm.social_link_id || undefined,
        platform: socialForm.platform,
        url: normalizedSocialURL,
        label: socialForm.label,
        position: Number(socialForm.position) || 0,
        is_visible: socialForm.is_visible,
      };

      if (socialForm.social_link_id) {
        await api.put("/profiles/social-links", payload);
      } else {
        await api.post("/profiles/social-links", payload);
      }

      await refreshMyProfile();
      setSocialForm({
        social_link_id: "",
        platform: socialForm.platform,
        url: "",
        label: "",
        position: 0,
        is_visible: true,
      });
      setNotice("Социальная ссылка сохранена.");
    } catch {
      setNotice("Не удалось сохранить социальную ссылку.");
    } finally {
      setSavingSocial(false);
    }
  };

  const handleEditSocialLink = (link: SocialLink) => {
    setSocialForm({
      social_link_id: link.social_link_id,
      platform: link.platform,
      url: link.url,
      label: link.label || "",
      position: link.position,
      is_visible: link.is_visible,
    });
  };

  const handleDeleteSocialLink = async (socialLinkID: string) => {
    setSavingSocial(true);
    setNotice(null);
    try {
      await api.delete(`/profiles/social-links/${socialLinkID}`);
      await refreshMyProfile();
      if (socialForm.social_link_id === socialLinkID) {
        setSocialForm({
          social_link_id: "",
          platform: "github",
          url: "",
          label: "",
          position: 0,
          is_visible: true,
        });
      }
      setNotice("Социальная ссылка удалена.");
    } catch {
      setNotice("Не удалось удалить социальную ссылку.");
    } finally {
      setSavingSocial(false);
    }
  };

  if (loading) {
    return <div className="space-y-4 animate-pulse"><Skeleton className="h-32 w-32 rounded-full" /></div>;
  }

  if (!profile) return <div>Пользователь не найден</div>;

  return (
    <div className="flex flex-col md:flex-row gap-8">
      <div className="w-full md:w-1/4 space-y-4">
        <Avatar className="w-full h-auto aspect-square">
          <AvatarImage src={profile.avatar_url || `https://github.com/identicons/${profile.username}.png`} />
          <AvatarFallback>{profile.username.substring(0, 2).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div>
          <h1 className="text-2xl font-bold">{displayName}</h1>
          <p className="text-sm text-muted-foreground">@{profile.username}</p>
          <p className="text-muted-foreground mt-2">{profile.bio || "Пользователь платформы"}</p>
          {profile.title?.label ? (
            <p className="mt-2 text-xs inline-block rounded-full border px-2 py-1 text-muted-foreground">
              {profile.title.label}
            </p>
          ) : null}
        </div>
        {!isSelf && user ? (
          <Button className="w-full" variant={isFollowing ? "outline" : "default"} onClick={handleFollowToggle} disabled={togglingFollow}>
            {togglingFollow ? "Обработка..." : isFollowing ? "Отписаться" : "Подписаться"}
          </Button>
        ) : null}
        <div className="flex gap-4 text-sm text-muted-foreground">
          <span className="flex items-center gap-1 hover:text-primary cursor-pointer">
            <Users className="h-4 w-4" /> {profile.followers_count} followers
          </span>
          <span className="hover:text-primary cursor-pointer">
            {profile.following_count} following
          </span>
        </div>
        {profile.location ? <p className="text-sm text-muted-foreground">Город: {profile.location}</p> : null}
        {profile.website_url ? (
          <a className="text-sm text-primary hover:underline" href={profile.website_url} target="_blank" rel="noreferrer">
            {profile.website_url}
          </a>
        ) : null}
        {notice ? <p className="text-xs text-muted-foreground">{notice}</p> : null}
      </div>

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
            {isSelf ? <TabsTrigger value="settings">Настройки профиля</TabsTrigger> : null}
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

          {isSelf ? (
            <TabsContent value="settings" className="space-y-8">
              <Card className="pt-2">
                <CardHeader className="space-y-4 py-4">
                  <CardTitle>Основная информация</CardTitle>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div className="space-y-1">
                      <Label htmlFor="display_name">Display Name</Label>
                      <Input
                        id="display_name"
                        value={profileForm.display_name}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, display_name: event.target.value }))}
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="location">Location</Label>
                      <Input
                        id="location"
                        value={profileForm.location}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, location: event.target.value }))}
                      />
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="bio">Bio</Label>
                      <Textarea
                        id="bio"
                        value={profileForm.bio}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, bio: event.target.value }))}
                        className="min-h-20"
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="avatar_url">Avatar URL</Label>
                      <Input
                        id="avatar_url"
                        value={profileForm.avatar_url}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, avatar_url: event.target.value }))}
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="cover_url">Cover URL</Label>
                      <Input
                        id="cover_url"
                        value={profileForm.cover_url}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, cover_url: event.target.value }))}
                      />
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="website_url">Website URL</Label>
                      <Input
                        id="website_url"
                        value={profileForm.website_url}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, website_url: event.target.value }))}
                      />
                    </div>
                  </div>
                  <div className="flex gap-3">
                    <Button onClick={handleSaveProfile} disabled={savingProfile}>
                      {savingProfile ? "Сохранение..." : "Сохранить профиль"}
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => setProfileForm((prev) => ({ ...prev, is_public: !prev.is_public }))}
                    >
                      {profileForm.is_public ? "Сделать приватным" : "Сделать публичным"}
                    </Button>
                  </div>
                </CardHeader>
              </Card>

              <Card className="pt-2">
                <CardHeader className="space-y-4 py-4">
                  <CardTitle>README профиля</CardTitle>
                  <Textarea
                    value={readmeMarkdown}
                    onChange={(event) => setReadmeMarkdown(event.target.value)}
                    className="min-h-40"
                    placeholder="# About me"
                  />
                  <Button onClick={handleSaveReadme} disabled={savingReadme}>
                    {savingReadme ? "Сохранение..." : "Сохранить README"}
                  </Button>
                </CardHeader>
              </Card>

              <Card className="pt-2">
                <CardHeader className="space-y-4 py-4">
                  <CardTitle>Титул профиля</CardTitle>
                  <div className="space-y-1 max-w-sm">
                    <Label htmlFor="title_code">Title Code</Label>
                    <Select value={titleCode} onValueChange={(value) => setTitleCode(value ?? "") }>
                      <SelectTrigger id="title_code" className="w-full">
                        <SelectValue placeholder="Выберите титул" />
                      </SelectTrigger>
                      <SelectContent>
                        {titles.map((title) => (
                          <SelectItem key={title.code} value={title.code}>
                            {title.label} ({title.code})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <Button onClick={handleSaveTitle} disabled={savingTitle || !titleCode}>
                    {savingTitle ? "Сохранение..." : "Установить титул"}
                  </Button>
                </CardHeader>
              </Card>

              <Card className="pt-2">
                <CardHeader className="space-y-4 py-4">
                  <CardTitle>Социальные ссылки</CardTitle>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div className="space-y-1">
                      <Label htmlFor="social_platform">Platform</Label>
                      <Select
                        value={socialForm.platform}
                        onValueChange={(value) => setSocialForm((prev) => ({ ...prev, platform: value ?? "github" }))}
                      >
                        <SelectTrigger id="social_platform" className="w-full">
                          <SelectValue placeholder="Выберите платформу" />
                        </SelectTrigger>
                        <SelectContent>
                          {SOCIAL_PLATFORMS.map((platform) => (
                            <SelectItem key={platform} value={platform}>
                              {platform}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="social_position">Position</Label>
                      <Input
                        id="social_position"
                        type="number"
                        value={socialForm.position}
                        onChange={(event) =>
                          setSocialForm((prev) => ({ ...prev, position: Number(event.target.value) || 0 }))
                        }
                      />
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="social_url">URL</Label>
                      <Input
                        id="social_url"
                        value={socialForm.url}
                        onChange={(event) => setSocialForm((prev) => ({ ...prev, url: event.target.value }))}
                      />
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="social_label">Label</Label>
                      <Input
                        id="social_label"
                        value={socialForm.label}
                        onChange={(event) => setSocialForm((prev) => ({ ...prev, label: event.target.value }))}
                      />
                    </div>
                  </div>
                  <div className="flex gap-3">
                    <Button onClick={handleSubmitSocialLink} disabled={savingSocial}>
                      {savingSocial ? "Сохранение..." : socialForm.social_link_id ? "Обновить ссылку" : "Добавить ссылку"}
                    </Button>
                    {socialForm.social_link_id ? (
                      <Button
                        variant="outline"
                        onClick={() =>
                          setSocialForm({
                            social_link_id: "",
                            platform: "github",
                            url: "",
                            label: "",
                            position: 0,
                            is_visible: true,
                          })
                        }
                      >
                        Сбросить
                      </Button>
                    ) : null}
                  </div>
                  <div className="space-y-2">
                    {(profile.social_links || []).map((link) => (
                      <div
                        key={link.social_link_id}
                        className="rounded-md border p-3 flex flex-col md:flex-row md:items-center md:justify-between gap-2"
                      >
                        <div className="min-w-0">
                          <p className="font-medium">
                            {link.platform}
                            {link.label ? ` • ${link.label}` : ""}
                          </p>
                          <a className="text-sm text-primary hover:underline break-all" href={link.url} target="_blank" rel="noreferrer">
                            {link.url}
                          </a>
                        </div>
                        <div className="flex gap-2">
                          <Button variant="outline" size="sm" onClick={() => handleEditSocialLink(link)}>
                            Изменить
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => void handleDeleteSocialLink(link.social_link_id)}
                            disabled={savingSocial}
                          >
                            Удалить
                          </Button>
                        </div>
                      </div>
                    ))}
                    {profile.social_links.length === 0 ? (
                      <p className="text-sm text-muted-foreground">Социальные ссылки пока не добавлены.</p>
                    ) : null}
                  </div>
                </CardHeader>
              </Card>
            </TabsContent>
          ) : null}
        </Tabs>
      </div>
    </div>
  );
}
