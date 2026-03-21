"use client";

import { useEffect, useState } from "react";
import { useRepo } from "@/components/repo/repo-context";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useRouter } from "next/navigation";
import { Repository } from "@/components/repo/repo-context";
import { getId, getErrorMessage } from "@/components/repo/repo-context";
import { toast } from "sonner";

export default function RepoSettingsPage() {
  const { repoId, isOwner, repo, fetchRepo } = useRepo();
  const router = useRouter();

  const [editRepoName, setEditRepoName] = useState("");
  const [editRepoDescription, setEditRepoDescription] = useState("");
  const [forks, setForks] = useState<Repository[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (repo) {
      setEditRepoName(repo.name || "");
      setEditRepoDescription(repo.description || "");
    }
  }, [repo]);

  useEffect(() => {
    if (!repoId) return;

    let isMounted = true;
    async function loadForks() {
      try {
        const res = await api.get(`/repositories/${repoId}/forks`);
        if (isMounted) setForks(res.data.repositories || []);
      } catch (err) {
        // Handle silently
      }
    }
    void loadForks();
    return () => { isMounted = false; };
  }, [repoId]);

  if (!isOwner) {
    return (
      <div className="p-8 text-center text-muted-foreground">
        У вас нет прав для доступа к настройкам репозитория.
      </div>
    );
  }

  const handleEditRepo = async () => {
    if (!repoId) return;

    const newName = editRepoName.trim();
    if (!newName) {
      toast.error("Название репозитория не может быть пустым");
      return;
    }

    try {
      setIsLoading(true);
      await api.patch(`/repositories/${repoId}`, { name: newName, description: editRepoDescription });
      toast.success("Настройки успешно обновлены!");
      await fetchRepo();
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка обновления"));
    } finally {
      setIsLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!repoId) return;
    if (!confirm("Вы уверены, что хотите удалить этот репозиторий? Это действие необратимо!")) return;

    try {
      await api.delete(`/repositories/${repoId}`);
      toast.success("Репозиторий удален");
      router.push("/");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "Ошибка при удалении"));
    }
  };

  return (
    <div className="max-w-4xl space-y-8 mt-6">
      <div>
        <h2 className="text-2xl font-semibold mb-4 pb-2 border-b">Settings</h2>
        
        <div className="space-y-4 max-w-xl">
          <div>
            <label className="block text-sm font-medium mb-1">Repository Name</label>
            <Input 
              value={editRepoName} 
              onChange={(e) => setEditRepoName(e.target.value)} 
              placeholder="repo-name" 
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Description</label>
            <Textarea 
              value={editRepoDescription} 
              onChange={(e) => setEditRepoDescription(e.target.value)} 
              rows={4}
              placeholder="Описание..." 
            />
          </div>
          <Button onClick={handleEditRepo} disabled={isLoading}>
            {isLoading ? "Saving..." : "Save changes"}
          </Button>
        </div>
      </div>

      <div>
        <h3 className="text-xl font-semibold mb-4 pb-2 border-b">Forks ({forks.length})</h3>
        {forks.length === 0 ? (
          <p className="text-sm text-muted-foreground">Пока нет форков.</p>
        ) : (
          <div className="space-y-2 border rounded-md divide-y">
            {forks.map((fork) => (
              <div key={fork.id || fork.repo_id || fork.slug} className="p-4 flex items-center justify-between">
                <div>
                  <button
                    type="button"
                    className="text-blue-500 font-medium hover:underline text-sm"
                    onClick={() => {
                      const forkOwner = fork.owner_username || fork.owner_id;
                      if (!forkOwner || !fork.slug) return;
                      router.push(`/${forkOwner}/${fork.slug}`);
                    }}
                  >
                    {fork.owner_username || fork.owner_id || "unknown"}/{fork.slug}
                  </button>
                  <p className="text-xs text-muted-foreground mt-1">{fork.name}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="border border-red-500 rounded-md overflow-hidden">
        <div className="bg-red-500/10 p-4 border-b border-red-500/20">
          <h3 className="text-red-500 font-semibold">Danger Zone</h3>
        </div>
        <div className="p-4 flex flex-col sm:flex-row gap-4 items-center justify-between">
          <div>
            <h4 className="font-medium">Delete this repository</h4>
            <p className="text-sm text-muted-foreground mt-1">
              Once you delete a repository, there is no going back. Please be certain.
            </p>
          </div>
          <Button variant="destructive" onClick={handleDelete}>
            Удалить репозиторий
          </Button>
        </div>
      </div>
    </div>
  );
}

