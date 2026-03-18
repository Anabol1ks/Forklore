"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { useAuthStore } from "@/store/auth";
import axios from "axios";

interface RepositoryResponse {
  slug?: string;
}

const cyrillicToLatinMap: Record<string, string> = {
  а: "a", б: "b", в: "v", г: "g", д: "d", е: "e", ё: "e",
  ж: "zh", з: "z", и: "i", й: "y", к: "k", л: "l", м: "m",
  н: "n", о: "o", п: "p", р: "r", с: "s", т: "t", у: "u",
  ф: "f", х: "h", ц: "ts", ч: "ch", ш: "sh", щ: "sch", ъ: "",
  ы: "y", ь: "", э: "e", ю: "yu", я: "ya",
};

function slugifyRepositoryName(value: string): string {
  const transliterated = value
    .trim()
    .toLowerCase()
    .split("")
    .map((ch) => cyrillicToLatinMap[ch] ?? ch)
    .join("");

  return transliterated
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function CreateRepoPage() {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState("public");
  const [type, setType] = useState("mixed");
  const [tagId, setTagId] = useState("");
  const [tags, setTags] = useState<{ tag_id: string; name: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const router = useRouter();
  const { user } = useAuthStore();
  const selectedTagName = useMemo(() => tags.find((tag) => tag.tag_id === tagId)?.name, [tags, tagId]);
  const slugPreview = useMemo(() => slugifyRepositoryName(name), [name]);

  useEffect(() => {
    const fetchTags = async () => {
      try {
        const res = await api.get("/repositories/tags");
        if (res.data?.tags) {
          setTags(res.data.tags);
        }
      } catch (err) {
        console.error("Failed to load tags:", err);
        toast.error("Не удалось загрузить теги репозиториев");
      }
    };
    fetchTags();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!user) {
      toast.error("Сначала войдите в систему");
      return;
    }
    if (!tagId) {
      toast.error("Выберите тег репозитория");
      return;
    }
    
    setLoading(true);
    try {
      const response = await api.post("/repositories", {
        name,
        slug: slugPreview,
        description,
        visibility,
        type,
        tag_id: tagId,
      });
      toast.success("Репозиторий создан!");
      const repository = (response.data as { repository?: RepositoryResponse }).repository;
      const repoSlug = repository?.slug || name;
      router.push(`/${user.username}/${repoSlug}`);
    } catch (error: unknown) {
      if (axios.isAxiosError(error)) {
        toast.error((error.response?.data as { message?: string } | undefined)?.message || "Ошибка при создании репозитория.");
      } else {
        toast.error("Ошибка при создании репозитория.");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto py-10 space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Создать новый репозиторий</h1>
        <p className="text-muted-foreground mt-2">
          Репозиторий содержит все материалы вашего проекта.
        </p>
      </div>

      <Card>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-6 pt-6">
            <div className="space-y-2">
              <Label htmlFor="name">Название репозитория *</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="algorithms-notes"
                required
              />
              <p className="text-xs text-muted-foreground">
                Будет создано как {user?.username || "username"}/{slugPreview || "repo"}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Описание</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Краткое описание материалов"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Видимость</Label>
                <Select value={visibility} onValueChange={(val) => setVisibility(val ?? "public")}>
                  <SelectTrigger>
                    <SelectValue placeholder="Выберите видимость" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="public">🌍 Публичный</SelectItem>
                    <SelectItem value="private">🔒 Приватный</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>Тип содержимого</Label>
                <Select value={type} onValueChange={(val) => setType(val ?? "mixed")}>
                  <SelectTrigger>
                    <SelectValue placeholder="Выберите тип" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="mixed">Смешанный (Mixed)</SelectItem>
                    <SelectItem value="article">Статьи (Article)</SelectItem>
                    <SelectItem value="notes">Конспекты (Notes)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label>Тег репозитория *</Label>
              <Select value={tagId} onValueChange={(val) => setTagId(val ?? "") }>
                <SelectTrigger>
                  <SelectValue placeholder={tags.length > 0 ? "Выберите тег" : "Загрузка тегов..."}>
                    {selectedTagName}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {tags.map((t) => (
                    <SelectItem key={t.tag_id} value={t.tag_id}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

          </CardContent>
          <CardFooter className="flex justify-between border-t px-6 py-4">
            <Button variant="outline" type="button" onClick={() => router.back()}>
              Отмена
            </Button>
            <Button type="submit" disabled={loading || !name || !tagId}>
              {loading ? "Создается..." : "Создать репозиторий"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
