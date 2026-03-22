"use client";

import { DragEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import Image from "next/image";
import { api } from "@/lib/api";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ArrowDownUp, Book, GripVertical, Star, Upload, Users } from "lucide-react";
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
  display_name?: string;
  avatar_url?: string;
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

const SOCIAL_PLATFORM_LABELS: Record<string, string> = {
  telegram: "Telegram",
  github: "GitHub",
  vk: "VK",
  linkedin: "LinkedIn",
  x: "X",
  youtube: "YouTube",
  website: "Сайт",
  other: "Другое",
};

const MAX_PROFILE_IMAGE_SIZE = 10 * 1024 * 1024;
const PROFILE_IMAGE_MIME_TYPES = ["image/jpeg", "image/png", "image/gif", "image/webp"];
const UNIVERSITY_OPTIONS = ["МИРЭА", "МГУ"] as const;

function validateProfileImageFile(file: File): string | null {
  if (!PROFILE_IMAGE_MIME_TYPES.includes(file.type)) {
    return "Неподдерживаемый формат. Разрешены JPEG, PNG, GIF, WebP.";
  }
  if (file.size > MAX_PROFILE_IMAGE_SIZE) {
    return "Файл слишком большой. Максимальный размер 10 МБ.";
  }
  return null;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} Б`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} КБ`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(2)} МБ`;
}

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
  const [followers, setFollowers] = useState<ProfilePreview[]>([]);
  const [following, setFollowing] = useState<ProfilePreview[]>([]);
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
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [uploadingCover, setUploadingCover] = useState(false);
  const [avatarDragActive, setAvatarDragActive] = useState(false);
  const [coverDragActive, setCoverDragActive] = useState(false);
  const [draggedSocialLinkID, setDraggedSocialLinkID] = useState<string | null>(null);
  const [dragOverSocialLinkID, setDragOverSocialLinkID] = useState<string | null>(null);
  const [activeDragHandleID, setActiveDragHandleID] = useState<string | null>(null);
  const [pendingAvatarFile, setPendingAvatarFile] = useState<File | null>(null);
  const [pendingCoverFile, setPendingCoverFile] = useState<File | null>(null);
  const [pendingAvatarPreviewURL, setPendingAvatarPreviewURL] = useState<string | null>(null);
  const [pendingCoverPreviewURL, setPendingCoverPreviewURL] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState("repositories");
  const [loading, setLoading] = useState(true);

  const avatarInputRef = useRef<HTMLInputElement | null>(null);
  const coverInputRef = useRef<HTMLInputElement | null>(null);

  const isSelf = useMemo(() => {
    if (!user || !profile) {
      return false;
    }
    return user.id === profile.user_id || user.username === profile.username;
  }, [profile, user]);

  const displayName = profile?.display_name?.trim() || profile?.username || ownerId;
  const coverBannerURL = pendingCoverPreviewURL || profile?.cover_url || "";
  const selectedUniversityValue = UNIVERSITY_OPTIONS.includes(profileForm.location as (typeof UNIVERSITY_OPTIONS)[number])
    ? profileForm.location
    : "other";

  useEffect(() => {
    if (!pendingAvatarFile) {
      setPendingAvatarPreviewURL(null);
      return;
    }

    const objectURL = URL.createObjectURL(pendingAvatarFile);
    setPendingAvatarPreviewURL(objectURL);
    return () => URL.revokeObjectURL(objectURL);
  }, [pendingAvatarFile]);

  useEffect(() => {
    if (!pendingCoverFile) {
      setPendingCoverPreviewURL(null);
      return;
    }

    const objectURL = URL.createObjectURL(pendingCoverFile);
    setPendingCoverPreviewURL(objectURL);
    return () => URL.revokeObjectURL(objectURL);
  }, [pendingCoverFile]);

  const sortedSocialLinks = useMemo(() => {
    if (!profile?.social_links) {
      return [];
    }

    return [...profile.social_links].sort((a, b) => {
      if (a.position !== b.position) {
        return a.position - b.position;
      }
      return a.platform.localeCompare(b.platform);
    });
  }, [profile?.social_links]);

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

  const refreshMyProfile = async (): Promise<UserProfile> => {
    const response = await api.get("/profiles/me");
    const nextProfile = response.data.profile as UserProfile;
    setProfile(nextProfile);
    applyProfileToForms(nextProfile);
    return nextProfile;
  };

  const persistSocialLinksOrder = async (links: SocialLink[]) => {
    for (let index = 0; index < links.length; index += 1) {
      const link = links[index];
      await api.put("/profiles/social-links", {
        social_link_id: link.social_link_id,
        platform: link.platform,
        url: link.url,
        label: link.label || "",
        position: index,
        is_visible: link.is_visible,
      });
    }
    await refreshMyProfile();
  };

  const uploadProfileImage = useCallback(async (kind: "avatar" | "cover", file: File): Promise<boolean> => {
    const validationError = validateProfileImageFile(file);
    if (validationError) {
      setNotice(validationError);
      return false;
    }

    if (kind === "avatar") {
      setUploadingAvatar(true);
    } else {
      setUploadingCover(true);
    }

    setNotice(null);
    try {
      const formData = new FormData();
      formData.append("kind", kind);
      formData.append("file", file);

      const response = await api.post("/profiles/me/image", formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });

      const nextProfile = response.data.profile as UserProfile;
      setProfile(nextProfile);
      applyProfileToForms(nextProfile);
      setNotice(kind === "avatar" ? "Аватар обновлён." : "Обложка обновлена.");
      return true;
    } catch {
      setNotice("Не удалось загрузить изображение. Проверьте формат и размер файла.");
      return false;
    } finally {
      if (kind === "avatar") {
        setUploadingAvatar(false);
      } else {
        setUploadingCover(false);
      }
    }
  }, []);

  const preparePendingImage = (kind: "avatar" | "cover", file: File | null) => {
    if (!file) {
      return;
    }

    const validationError = validateProfileImageFile(file);
    if (validationError) {
      setNotice(validationError);
      return;
    }

    if (kind === "avatar") {
      setPendingAvatarFile(file);
    } else {
      setPendingCoverFile(file);
    }

    setNotice(`Файл готов к загрузке: ${file.name} (${formatFileSize(file.size)}).`);
  };

  const clearPendingImage = (kind: "avatar" | "cover") => {
    if (kind === "avatar") {
      setPendingAvatarFile(null);
    } else {
      setPendingCoverFile(null);
    }
  };

  const commitPendingUpload = async (kind: "avatar" | "cover") => {
    const file = kind === "avatar" ? pendingAvatarFile : pendingCoverFile;
    if (!file) {
      return;
    }

    const success = await uploadProfileImage(kind, file);
    if (success) {
      clearPendingImage(kind);
    }
  };

  const handleFileInputChange = (kind: "avatar" | "cover", file: File | null) => {
    preparePendingImage(kind, file);
  };

  const handleDropImage = (kind: "avatar" | "cover", event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (kind === "avatar") {
      setAvatarDragActive(false);
    } else {
      setCoverDragActive(false);
    }

    const file = event.dataTransfer.files?.[0] || null;
    if (!file) {
      return;
    }

    preparePendingImage(kind, file);
  };

  const handleSocialDragStart = (event: DragEvent<HTMLDivElement>, socialLinkID: string) => {
    if (activeDragHandleID !== socialLinkID) {
      event.preventDefault();
      return;
    }

    event.dataTransfer.effectAllowed = "move";
    setDraggedSocialLinkID(socialLinkID);
  };

  const handleSocialDrop = async (targetSocialLinkID: string) => {
    if (!profile || !draggedSocialLinkID || draggedSocialLinkID === targetSocialLinkID) {
      setDraggedSocialLinkID(null);
      return;
    }

    const current = [...sortedSocialLinks];
    const sourceIndex = current.findIndex((item) => item.social_link_id === draggedSocialLinkID);
    const targetIndex = current.findIndex((item) => item.social_link_id === targetSocialLinkID);

    if (sourceIndex < 0 || targetIndex < 0) {
      setDraggedSocialLinkID(null);
      return;
    }

    const [moved] = current.splice(sourceIndex, 1);
    current.splice(targetIndex, 0, moved);

    const ordered = current.map((item, index) => ({ ...item, position: index }));
    setProfile((prev) => (prev ? { ...prev, social_links: ordered } : prev));

    setNotice("Сохраняем новый порядок ссылок...");
    try {
      await persistSocialLinksOrder(ordered);
      setNotice("Порядок социальных ссылок обновлён.");
    } catch {
      setNotice("Не удалось сохранить порядок социальных ссылок.");
      await refreshMyProfile();
    } finally {
      setDraggedSocialLinkID(null);
      setDragOverSocialLinkID(null);
    }
  };

  const loadRelations = useCallback(async (targetProfile: UserProfile) => {
    try {
      const [followersResponse, followingResponse] = await Promise.all([
        api.get<ListResponse<ProfilePreview>>(`/profiles/${targetProfile.user_id}/followers`, {
          params: { limit: 100, offset: 0 },
        }),
        api.get<ListResponse<ProfilePreview>>(`/profiles/${targetProfile.user_id}/following`, {
          params: { limit: 100, offset: 0 },
        }),
      ]);

      setFollowers(followersResponse.data.profiles || []);
      setFollowing(followingResponse.data.profiles || []);
    } catch {
      setFollowers([]);
      setFollowing([]);
    }
  }, []);

  const loadProfileSocialLinks = useCallback(async (targetProfile: UserProfile) => {
    try {
      const response = await api.get(`/profiles/${targetProfile.user_id}/social-links`);
      const links = (response.data.social_links || []) as SocialLink[];
      setProfile((prev) => {
        if (!prev) {
          return prev;
        }
        return {
          ...prev,
          social_links: links,
        };
      });
    } catch {
      // Keep social links from profile payload if dedicated endpoint fails.
    }
  }, []);

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

        await Promise.all([
          loadFollowState(profileResult),
          loadRelations(profileResult),
          loadProfileSocialLinks(profileResult),
        ]);
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
  }, [loadFollowState, loadProfile, loadProfileSocialLinks, loadRelations, loadTitles, ownerId, user]);

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

      await loadRelations(profile);
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
        await refreshMyProfile();
      } else {
        const response = await api.post("/profiles/social-links", { ...payload, position: 0 });
        const created = response.data.social_link as SocialLink;
        const updatedProfile = await refreshMyProfile();

        if (created?.social_link_id) {
          const links = [...(updatedProfile.social_links || [])];
          const createdIndex = links.findIndex((item) => item.social_link_id === created.social_link_id);
          if (createdIndex > 0) {
            const [createdLink] = links.splice(createdIndex, 1);
            links.unshift(createdLink);
            await persistSocialLinksOrder(links);
          }
        }
      }

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
    <div className="space-y-6">
      <div className="relative h-44 w-full overflow-hidden rounded-xl border md:h-56">
        {coverBannerURL ? (
          <div
            className="absolute inset-0 bg-cover bg-center"
            style={{ backgroundImage: `url(${coverBannerURL})` }}
          />
        ) : (
          <div className="absolute inset-0 bg-gradient-to-r from-slate-200 via-zinc-100 to-stone-200 dark:from-slate-800 dark:via-zinc-900 dark:to-stone-800" />
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-black/45 via-black/10 to-transparent" />
        <div className="absolute bottom-3 left-4 right-4 flex items-end justify-between text-white">
          <div className="min-w-0">
            <p className="truncate text-lg font-semibold md:text-xl">{displayName}</p>
            <p className="truncate text-xs text-white/85 md:text-sm">@{profile.username}</p>
          </div>
          {profile.title?.label ? (
            <span className="ml-3 rounded-full border border-white/50 bg-black/20 px-2 py-1 text-xs">
              {profile.title.label}
            </span>
          ) : null}
        </div>
      </div>

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
        <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
          <span className="flex items-center gap-1 hover:text-primary cursor-pointer" onClick={() => setActiveTab("followers")}>
            <Users className="h-4 w-4" /> {profile.followers_count} подписчиков
          </span>
          <span className="hover:text-primary cursor-pointer" onClick={() => setActiveTab("following")}>
            {profile.following_count} подписок
          </span>
        </div>
        {profile.location ? <p className="text-sm text-muted-foreground">Вуз: {profile.location}</p> : null}
        {profile.website_url ? (
          <a className="text-sm text-primary hover:underline" href={profile.website_url} target="_blank" rel="noreferrer">
            {profile.website_url}
          </a>
        ) : null}
        {sortedSocialLinks.length > 0 ? (
          <div className="space-y-2">
            <p className="text-sm font-medium">Социальные сети</p>
            <div className="space-y-1">
              {sortedSocialLinks.map((link) => (
                <a
                  key={link.social_link_id}
                  className="block text-sm text-primary hover:underline break-all"
                  href={link.url}
                  target="_blank"
                  rel="noreferrer"
                >
                  {SOCIAL_PLATFORM_LABELS[link.platform] || link.platform}
                  {link.label ? ` • ${link.label}` : ""}
                </a>
              ))}
            </div>
          </div>
        ) : null}
        {notice ? <p className="text-xs text-muted-foreground">{notice}</p> : null}
      </div>

      <div className="w-full md:w-3/4">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="mb-4 w-full">
            <TabsTrigger value="repositories" className="flex items-center gap-2">
              <Book className="h-4 w-4" /> Репозитории
              <span className="ml-1 rounded-full bg-muted px-2 py-0.5 text-xs">{repos.length}</span>
            </TabsTrigger>
            <TabsTrigger value="stars" className="flex items-center gap-2">
              <Star className="h-4 w-4" /> Избранное
            </TabsTrigger>
            <TabsTrigger value="followers" className="flex items-center gap-2">
              <Users className="h-4 w-4" /> Подписчики
              <span className="ml-1 rounded-full bg-muted px-2 py-0.5 text-xs">{followers.length}</span>
            </TabsTrigger>
            <TabsTrigger value="following" className="flex items-center gap-2">
              <Users className="h-4 w-4" /> Подписки
              <span className="ml-1 rounded-full bg-muted px-2 py-0.5 text-xs">{following.length}</span>
            </TabsTrigger>
            {isSelf ? <TabsTrigger value="settings">Настройки профиля</TabsTrigger> : null}
          </TabsList>

          <TabsContent value="repositories" className="space-y-4">
            <Input placeholder="Найти репозиторий..." className="w-full max-w-md" />
            <div className="grid grid-cols-1 gap-4">
              {repos.map((repo) => (
                <Card key={repo.repo_id || repo.id || repo.name} className="pt-2 hover:border-primary/50 transition">
                  <CardHeader className="py-4">
                    <div className="flex items-start justify-between gap-2">
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
                    <div className="flex items-start justify-between gap-2">
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

          <TabsContent value="followers" className="space-y-3">
            {followers.length > 0 ? followers.map((item) => (
              <Card key={`${item.user_id}-${item.username}`} className="pt-2 hover:border-primary/50 transition">
                <CardHeader className="py-4">
                  <Link href={`/user/${item.username || item.user_id}`} className="hover:underline">
                    <CardTitle className="text-base text-primary">
                      {item.display_name || item.username}
                    </CardTitle>
                  </Link>
                  <CardDescription>@{item.username}</CardDescription>
                </CardHeader>
              </Card>
            )) : (
              <div className="p-10 text-center border rounded-md text-muted-foreground">
                Подписчиков пока нет.
              </div>
            )}
          </TabsContent>

          <TabsContent value="following" className="space-y-3">
            {following.length > 0 ? following.map((item) => (
              <Card key={`${item.user_id}-${item.username}`} className="pt-2 hover:border-primary/50 transition">
                <CardHeader className="py-4">
                  <Link href={`/user/${item.username || item.user_id}`} className="hover:underline">
                    <CardTitle className="text-base text-primary">
                      {item.display_name || item.username}
                    </CardTitle>
                  </Link>
                  <CardDescription>@{item.username}</CardDescription>
                </CardHeader>
              </Card>
            )) : (
              <div className="p-10 text-center border rounded-md text-muted-foreground">
                Пользователь пока ни на кого не подписан.
              </div>
            )}
          </TabsContent>

          {isSelf ? (
            <TabsContent value="settings" className="space-y-8">
              <Card className="pt-2">
                <CardHeader className="space-y-4 py-4">
                  <CardTitle>Основная информация</CardTitle>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div className="space-y-1">
                      <Label htmlFor="display_name">Отображаемое имя</Label>
                      <Input
                        id="display_name"
                        value={profileForm.display_name}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, display_name: event.target.value }))}
                        placeholder="Как вас показывать на платформе"
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="university">Вуз</Label>
                      <Select
                        value={selectedUniversityValue}
                        onValueChange={(value) => {
                          if (value === "other") {
                            setProfileForm((prev) => ({ ...prev, location: "" }));
                            return;
                          }
                          setProfileForm((prev) => ({ ...prev, location: value ?? "" }));
                        }}
                      >
                        <SelectTrigger id="university" className="w-full">
                          <SelectValue placeholder="Выберите вуз" />
                        </SelectTrigger>
                        <SelectContent>
                          {UNIVERSITY_OPTIONS.map((university) => (
                            <SelectItem key={university} value={university}>
                              {university}
                            </SelectItem>
                          ))}
                          <SelectItem value="other">Другой</SelectItem>
                        </SelectContent>
                      </Select>
                      {selectedUniversityValue === "other" ? (
                        <Input
                          id="location"
                          value={profileForm.location}
                          onChange={(event) => setProfileForm((prev) => ({ ...prev, location: event.target.value }))}
                          placeholder="Укажите ваш вуз"
                        />
                      ) : null}
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="bio">О себе</Label>
                      <Textarea
                        id="bio"
                        value={profileForm.bio}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, bio: event.target.value }))}
                        className="min-h-20"
                        placeholder="Коротко расскажите о себе"
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="avatar_url">Ссылка на аватар</Label>
                      <Input
                        id="avatar_url"
                        value={profileForm.avatar_url}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, avatar_url: event.target.value }))}
                        placeholder="https://..."
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="cover_url">Ссылка на обложку</Label>
                      <Input
                        id="cover_url"
                        value={profileForm.cover_url}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, cover_url: event.target.value }))}
                        placeholder="https://..."
                      />
                    </div>
                    <div className="space-y-2 md:col-span-2">
                      <Label>Загрузка аватара файлом</Label>
                      <div
                        className={`rounded-lg border border-dashed p-4 transition ${avatarDragActive ? "border-primary bg-primary/5" : "border-border"}`}
                        onDragOver={(event) => {
                          event.preventDefault();
                          setAvatarDragActive(true);
                        }}
                        onDragLeave={() => setAvatarDragActive(false)}
                        onDrop={(event) => handleDropImage("avatar", event)}
                      >
                        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                          <p className="text-sm text-muted-foreground">
                            Перетащите изображение сюда или выберите файл. Как на GitHub: JPEG, PNG, GIF, WebP до 10 МБ.
                          </p>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="w-fit"
                            onClick={() => avatarInputRef.current?.click()}
                            disabled={uploadingAvatar}
                          >
                            <Upload className="mr-2 h-4 w-4" />
                            {uploadingAvatar ? "Загрузка..." : "Выбрать файл"}
                          </Button>
                        </div>
                        <input
                          ref={avatarInputRef}
                          type="file"
                          accept="image/jpeg,image/png,image/gif,image/webp"
                          className="hidden"
                          onChange={(event) => handleFileInputChange("avatar", event.target.files?.[0] || null)}
                        />
                        {pendingAvatarFile ? (
                          <div className="mt-3 rounded-md border bg-background p-3 space-y-2">
                            <p className="text-sm font-medium">Предпросмотр аватара</p>
                            {pendingAvatarPreviewURL ? (
                              <Image
                                src={pendingAvatarPreviewURL}
                                alt="Предпросмотр аватара"
                                width={96}
                                height={96}
                                unoptimized
                                className="h-24 w-24 rounded-full object-cover border"
                              />
                            ) : null}
                            <p className="text-xs text-muted-foreground">
                              {pendingAvatarFile.name} ({formatFileSize(pendingAvatarFile.size)})
                            </p>
                            <div className="flex flex-wrap gap-2">
                              <Button
                                type="button"
                                size="sm"
                                onClick={() => void commitPendingUpload("avatar")}
                                disabled={uploadingAvatar}
                              >
                                {uploadingAvatar ? "Загрузка..." : "Загрузить аватар"}
                              </Button>
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => clearPendingImage("avatar")}
                                disabled={uploadingAvatar}
                              >
                                Отменить
                              </Button>
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </div>
                    <div className="space-y-2 md:col-span-2">
                      <Label>Загрузка обложки файлом</Label>
                      <div
                        className={`rounded-lg border border-dashed p-4 transition ${coverDragActive ? "border-primary bg-primary/5" : "border-border"}`}
                        onDragOver={(event) => {
                          event.preventDefault();
                          setCoverDragActive(true);
                        }}
                        onDragLeave={() => setCoverDragActive(false)}
                        onDrop={(event) => handleDropImage("cover", event)}
                      >
                        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                          <p className="text-sm text-muted-foreground">
                            Поддерживаются те же ограничения: JPEG, PNG, GIF, WebP до 10 МБ.
                          </p>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="w-fit"
                            onClick={() => coverInputRef.current?.click()}
                            disabled={uploadingCover}
                          >
                            <Upload className="mr-2 h-4 w-4" />
                            {uploadingCover ? "Загрузка..." : "Выбрать файл"}
                          </Button>
                        </div>
                        <input
                          ref={coverInputRef}
                          type="file"
                          accept="image/jpeg,image/png,image/gif,image/webp"
                          className="hidden"
                          onChange={(event) => handleFileInputChange("cover", event.target.files?.[0] || null)}
                        />
                        {pendingCoverFile ? (
                          <div className="mt-3 rounded-md border bg-background p-3 space-y-2">
                            <p className="text-sm font-medium">Предпросмотр обложки</p>
                            {pendingCoverPreviewURL ? (
                              <Image
                                src={pendingCoverPreviewURL}
                                alt="Предпросмотр обложки"
                                width={640}
                                height={96}
                                unoptimized
                                className="h-24 w-full rounded object-cover border"
                              />
                            ) : null}
                            <p className="text-xs text-muted-foreground">
                              {pendingCoverFile.name} ({formatFileSize(pendingCoverFile.size)})
                            </p>
                            <div className="flex flex-wrap gap-2">
                              <Button
                                type="button"
                                size="sm"
                                onClick={() => void commitPendingUpload("cover")}
                                disabled={uploadingCover}
                              >
                                {uploadingCover ? "Загрузка..." : "Загрузить обложку"}
                              </Button>
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => clearPendingImage("cover")}
                                disabled={uploadingCover}
                              >
                                Отменить
                              </Button>
                            </div>
                          </div>
                        ) : null}
                      </div>
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="website_url">Сайт</Label>
                      <Input
                        id="website_url"
                        value={profileForm.website_url}
                        onChange={(event) => setProfileForm((prev) => ({ ...prev, website_url: event.target.value }))}
                        placeholder="https://ваш-сайт.example"
                      />
                    </div>
                  </div>
                  <div className="flex flex-col gap-3 sm:flex-row">
                    <Button className="w-full sm:w-auto" onClick={handleSaveProfile} disabled={savingProfile}>
                      {savingProfile ? "Сохранение..." : "Сохранить профиль"}
                    </Button>
                    <Button
                      variant="outline"
                      className="w-full sm:w-auto"
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
                    <Label htmlFor="title_code">Титул</Label>
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
                      <Label htmlFor="social_platform">Платформа</Label>
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
                              {SOCIAL_PLATFORM_LABELS[platform] || platform}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="social_visibility">Видимость</Label>
                      <Select
                        value={socialForm.is_visible ? "visible" : "hidden"}
                        onValueChange={(value) =>
                          setSocialForm((prev) => ({ ...prev, is_visible: (value ?? "visible") === "visible" }))
                        }
                      >
                        <SelectTrigger id="social_visibility" className="w-full">
                          <SelectValue placeholder="Выберите видимость" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="visible">Показывать в профиле</SelectItem>
                          <SelectItem value="hidden">Скрыть из публичного профиля</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="social_url">Ссылка</Label>
                      <Input
                        id="social_url"
                        value={socialForm.url}
                        onChange={(event) => setSocialForm((prev) => ({ ...prev, url: event.target.value }))}
                        placeholder="https://... или @username"
                      />
                    </div>
                    <div className="space-y-1 md:col-span-2">
                      <Label htmlFor="social_label">Подпись (необязательно)</Label>
                      <Input
                        id="social_label"
                        value={socialForm.label}
                        onChange={(event) => setSocialForm((prev) => ({ ...prev, label: event.target.value }))}
                        placeholder="Например: Основной аккаунт"
                      />
                    </div>
                  </div>
                  <div className="rounded-md border bg-muted/30 p-3 text-sm text-muted-foreground">
                    <p className="flex items-center gap-2">
                      <ArrowDownUp className="h-4 w-4" />
                      Новая ссылка автоматически становится первой. Порядок можно менять перетаскиванием.
                    </p>
                  </div>
                  <div className="flex flex-col gap-3 sm:flex-row">
                    <Button className="w-full sm:w-auto" onClick={handleSubmitSocialLink} disabled={savingSocial}>
                      {savingSocial ? "Сохранение..." : socialForm.social_link_id ? "Обновить ссылку" : "Добавить ссылку"}
                    </Button>
                    {socialForm.social_link_id ? (
                      <Button
                        variant="outline"
                        className="w-full sm:w-auto"
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
                    {sortedSocialLinks.map((link) => (
                      <div
                        key={link.social_link_id}
                        className={`rounded-md border p-3 flex flex-col md:flex-row md:items-center md:justify-between gap-2 transition-all duration-200 ${draggedSocialLinkID === link.social_link_id ? "opacity-60 scale-[0.99]" : "opacity-100 scale-100"} ${dragOverSocialLinkID === link.social_link_id ? "border-primary bg-primary/5" : "border-border"}`}
                        draggable={activeDragHandleID === link.social_link_id}
                        onDragStart={(event) => handleSocialDragStart(event, link.social_link_id)}
                        onDragEnter={() => {
                          if (draggedSocialLinkID && draggedSocialLinkID !== link.social_link_id) {
                            setDragOverSocialLinkID(link.social_link_id);
                          }
                        }}
                        onDragOver={(event) => {
                          event.preventDefault();
                          if (draggedSocialLinkID && draggedSocialLinkID !== link.social_link_id) {
                            setDragOverSocialLinkID(link.social_link_id);
                          }
                        }}
                        onDrop={() => void handleSocialDrop(link.social_link_id)}
                        onDragEnd={() => {
                          setDraggedSocialLinkID(null);
                          setDragOverSocialLinkID(null);
                          setActiveDragHandleID(null);
                        }}
                      >
                        <div className="min-w-0 flex items-start gap-2">
                          <button
                            type="button"
                            className="mt-0.5 rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground cursor-grab active:cursor-grabbing"
                            onPointerDown={() => setActiveDragHandleID(link.social_link_id)}
                            onPointerUp={() => setActiveDragHandleID(link.social_link_id)}
                            aria-label="Перетащить ссылку"
                            title="Перетащить"
                          >
                            <GripVertical className="h-4 w-4" />
                          </button>
                          <div className="min-w-0">
                            <p className="font-medium">
                              {SOCIAL_PLATFORM_LABELS[link.platform] || link.platform}
                              {link.label ? ` • ${link.label}` : ""}
                            </p>
                            <a className="text-sm text-primary hover:underline break-all" href={link.url} target="_blank" rel="noreferrer">
                              {link.url}
                            </a>
                            {!link.is_visible ? <p className="text-xs text-muted-foreground mt-1">Скрыта из публичного профиля</p> : null}
                          </div>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          <p className="font-medium">
                            Позиция: {link.position + 1}
                          </p>
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
                    {sortedSocialLinks.length === 0 ? (
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
    </div>
  );
}
