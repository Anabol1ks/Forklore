"use client";

import { useState } from "react";
import { useRepo } from "@/components/repo/repo-context";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";
import { getErrorMessage } from "@/components/repo/repo-context";
import { toast } from "sonner";
import { ArrowLeft, UploadCloud } from "lucide-react";
import { Input } from "@/components/ui/input";

export default function UploadFilePage() {
  const { repoId, isOwner, repo } = useRepo();
  const router = useRouter();

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!isOwner) {
    return <div className="p-8 text-center">Нет доступа</div>;
  }

  const handleUpload = async () => {
    if (!selectedFile) {
      toast.error("Выберите файл");
      return;
    }

    try {
      setIsSubmitting(true);
      const formData = new FormData();
      formData.append("file", selectedFile);

      await api.post(`/repositories/${repoId}/files/upload`, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });

      toast.success("Файл успешно загружен!");
      router.push(`/${repo?.owner_username || "owner"}/${repo?.slug}`);
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка загрузки файла"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const owner = repo?.owner_username || repo?.owner_id || "owner";
  const slug = repo?.slug || "repo";

  return (
    <div className="max-w-3xl mx-auto space-y-6 mt-4">
      <div className="flex items-center gap-4 border-b pb-4">
        <Button variant="ghost" size="icon" onClick={() => router.push(`/${owner}/${slug}`)}><ArrowLeft className="h-5 w-5" /></Button>
        <h2 className="text-xl font-semibold">Upload files</h2>
      </div>

      <div className="border-2 border-dashed rounded-lg p-12 text-center flex flex-col items-center justify-center space-y-4">
          <UploadCloud className="h-10 w-10 text-muted-foreground" />
          <div>
            <p className="text-lg font-medium">Choose your files</p>
            <p className="text-sm text-muted-foreground">Select a file from your computer to upload to the repository.</p>
          </div>
          <Input 
            type="file" 
            className="max-w-xs"
            onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
          />
      </div>

      <div className="flex justify-end gap-2">
         <Button variant="outline" onClick={() => router.push(`/${owner}/${slug}`)}>Cancel</Button>
         <Button onClick={handleUpload} disabled={isSubmitting || !selectedFile}>
            {isSubmitting ? "Uploading..." : "Commit changes"}
         </Button>
      </div>
    </div>
  );
}



