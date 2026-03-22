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

type RepositoryTag = {
  tag_id: string;
  name: string;
};

type UniversityKey = "МИРЭА" | "МГУ";

const UNIVERSITY_KEYS: UniversityKey[] = ["МИРЭА", "МГУ"];
const QUICK_SUBJECT_LIMIT = 8;

function stripUniversityPrefix(tagName: string): string {
  return tagName.replace(/^МИРЭА •\s*/, "").replace(/^МГУ •\s*/, "");
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
  const [tags, setTags] = useState<RepositoryTag[]>([]);
  const [selectedUniversity, setSelectedUniversity] = useState<UniversityKey | "">("");
  const [subjectQuery, setSubjectQuery] = useState("");
  const [activeSubjectIndex, setActiveSubjectIndex] = useState(-1);
  const [loading, setLoading] = useState(false);
  const router = useRouter();
  const { user } = useAuthStore();
  const selectedTagName = useMemo(() => tags.find((tag) => tag.tag_id === tagId)?.name, [tags, tagId]);
  const slugPreview = useMemo(() => slugifyRepositoryName(name), [name]);

  const universityTags = useMemo(() => {
    const groups: Record<UniversityKey, RepositoryTag[]> = {
      МИРЭА: [],
      МГУ: [],
    };

    for (const tag of tags) {
      if (tag.name.startsWith("МИРЭА • ")) {
        groups.МИРЭА.push(tag);
      } else if (tag.name.startsWith("МГУ • ")) {
        groups.МГУ.push(tag);
      }
    }

    return groups;
  }, [tags]);

  const selectedUniversityTags = useMemo(() => {
    if (!selectedUniversity) {
      return [];
    }

    return universityTags[selectedUniversity];
  }, [selectedUniversity, universityTags]);

  const filteredSubjects = useMemo(() => {
    const query = subjectQuery.trim().toLowerCase();
    if (!query) {
      return selectedUniversityTags;
    }

    return selectedUniversityTags.filter((tag) => tag.name.toLowerCase().includes(query));
  }, [selectedUniversityTags, subjectQuery]);

  const quickSubjects = useMemo(() => filteredSubjects.slice(0, QUICK_SUBJECT_LIMIT), [filteredSubjects]);

  useEffect(() => {
    if (filteredSubjects.length === 0) {
      setActiveSubjectIndex(-1);
      return;
    }

    setActiveSubjectIndex((prev) => {
      if (prev < 0) {
        return 0;
      }
      if (prev >= filteredSubjects.length) {
        return filteredSubjects.length - 1;
      }
      return prev;
    });
  }, [filteredSubjects]);

  useEffect(() => {
    const fetchTags = async () => {
      try {
        const res = await api.get("/repositories/tags");
        if (res.data?.tags) {
          const nextTags = (res.data.tags as RepositoryTag[]) || [];
          setTags(nextTags);
        }
      } catch (err) {
        console.error("Failed to load tags:", err);
        toast.error("Не удалось загрузить теги репозиториев");
      }
    };
    fetchTags();
  }, []);

  useEffect(() => {
    const selected = tags.find((tag) => tag.tag_id === tagId);
    if (!selected) {
      return;
    }
    if (selected.name.startsWith("МИРЭА • ")) {
      setSelectedUniversity("МИРЭА");
    } else if (selected.name.startsWith("МГУ • ")) {
      setSelectedUniversity("МГУ");
    }
  }, [tagId, tags]);

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
              <Label>Вуз *</Label>
              <Select
                value={selectedUniversity}
                onValueChange={(val) => {
                  const nextUniversity = (val as UniversityKey) || "";
                  setSelectedUniversity(nextUniversity);
                  setSubjectQuery("");
                  setTagId("");
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Выберите университет" />
                </SelectTrigger>
                <SelectContent>
                  {UNIVERSITY_KEYS.map((university) => (
                    <SelectItem key={university} value={university}>
                      {university}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="subject-query">Поиск предмета</Label>
              <Input
                id="subject-query"
                value={subjectQuery}
                onChange={(event) => setSubjectQuery(event.target.value)}
                onKeyDown={(event) => {
                  if (!selectedUniversity || filteredSubjects.length === 0) {
                    return;
                  }

                  if (event.key === "ArrowDown") {
                    event.preventDefault();
                    setActiveSubjectIndex((prev) => {
                      const next = prev < filteredSubjects.length - 1 ? prev + 1 : 0;
                      setTagId(filteredSubjects[next].tag_id);
                      return next;
                    });
                    return;
                  }

                  if (event.key === "ArrowUp") {
                    event.preventDefault();
                    setActiveSubjectIndex((prev) => {
                      const next = prev > 0 ? prev - 1 : filteredSubjects.length - 1;
                      setTagId(filteredSubjects[next].tag_id);
                      return next;
                    });
                    return;
                  }

                  if (event.key === "Enter") {
                    event.preventDefault();
                    const current = activeSubjectIndex >= 0 ? activeSubjectIndex : 0;
                    const selected = filteredSubjects[current];
                    if (selected) {
                      setTagId(selected.tag_id);
                    }
                  }
                }}
                placeholder={selectedUniversity ? "Начните вводить предмет" : "Сначала выберите вуз"}
                disabled={!selectedUniversity}
              />
              {selectedUniversity && quickSubjects.length > 0 ? (
                <div className="flex flex-wrap gap-2 pt-1">
                  {quickSubjects.map((subject) => (
                    <Button
                      key={`quick-${subject.tag_id}`}
                      type="button"
                      size="sm"
                      variant={tagId === subject.tag_id ? "default" : "outline"}
                      onClick={() => setTagId(subject.tag_id)}
                    >
                      {stripUniversityPrefix(subject.name)}
                    </Button>
                  ))}
                </div>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label>Предмет (тег репозитория) *</Label>
              {!selectedUniversity ? (
                <div className="rounded-md border border-dashed px-3 py-6 text-sm text-muted-foreground">
                  Сначала выберите вуз, затем начните вводить название предмета.
                </div>
              ) : null}

              {selectedUniversity ? (
                <div className="rounded-md border bg-card">
                  <div className="max-h-64 overflow-y-auto p-2 space-y-1">
                    {filteredSubjects.length === 0 ? (
                      <p className="px-2 py-3 text-sm text-muted-foreground">По вашему запросу предметы не найдены.</p>
                    ) : (
                      filteredSubjects.map((subject, index) => {
                        const isSelected = tagId === subject.tag_id;
                        const isActive = activeSubjectIndex === index;

                        return (
                          <button
                            key={subject.tag_id}
                            type="button"
                            className={`w-full rounded-md px-3 py-2 text-left text-sm transition ${
                              isSelected
                                ? "bg-primary text-primary-foreground"
                                : isActive
                                  ? "bg-muted"
                                  : "hover:bg-muted"
                            }`}
                            onMouseEnter={() => setActiveSubjectIndex(index)}
                            onClick={() => {
                              setActiveSubjectIndex(index);
                              setTagId(subject.tag_id);
                            }}
                          >
                            {stripUniversityPrefix(subject.name)}
                          </button>
                        );
                      })
                    )}
                  </div>
                </div>
              ) : null}

              <p className="text-sm text-muted-foreground">
                {selectedTagName
                  ? `Выбран предмет: ${stripUniversityPrefix(selectedTagName)}`
                  : "Предмет пока не выбран."}
              </p>
              {selectedUniversity ? (
                <p className="text-xs text-muted-foreground">
                  Выбрано: {selectedUniversity}. Доступно предметов: {selectedUniversityTags.length}.
                </p>
              ) : null}
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
