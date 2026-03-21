"use client";

import { useState } from "react";
import { useRepo } from "@/components/repo/repo-context";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useRouter } from "next/navigation";
import { getErrorMessage } from "@/components/repo/repo-context";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import { MarkdownPreview } from "@/components/markdown/markdown-preview";

export default function NewDocumentPage() {
  const { repoId, isOwner, repo } = useRepo();
  const router = useRouter();

  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<"edit" | "preview">("edit");

  if (!isOwner) {
    return <div className="p-8 text-center">Нет доступа</div>;
  }

  const handleCreate = async () => {
    if (!title.trim()) {
      toast.error("Введите название документа");
      return;
    }

    try {
      setIsSubmitting(true);
      await api.post(`/repositories/${repoId}/documents`, {
        title: title.trim(),
        initial_content: content,
      });
      toast.success("Документ создан!");
      router.push(`/${repo?.owner_username || "owner"}/${repo?.slug}`);
    } catch (error) {
      toast.error(getErrorMessage(error, "Ошибка при создании документа"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const owner = repo?.owner_username || repo?.owner_id || "owner";
  const slug = repo?.slug || "repo";

  return (
    <div className="max-w-5xl mx-auto space-y-6 mt-4">
      <div className="flex items-center gap-4 border-b pb-4">
        <Button variant="ghost" size="icon" onClick={() => router.push(`/${owner}/${slug}`)}><ArrowLeft className="h-5 w-5" /></Button>
        <h2 className="text-xl font-semibold">Create new Document</h2>
      </div>

      <div className="bg-card border rounded-md overflow-hidden flex flex-col items-stretch space-y-0 relative">
         <div className="bg-muted px-4 py-2 border-b w-full flex items-center justify-between">
           <Input 
             value={title}
             onChange={(e) => setTitle(e.target.value)}
             placeholder="Name your document..."
             className="max-w-md h-8 font-medium bg-background"
           />
         </div>
         
         <div className="bg-muted px-4 border-b flex gap-2 w-full pt-2">
            <button 
              className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${viewMode === 'edit' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              onClick={() => setViewMode('edit')}
            >
              Edit
            </button>
            <button 
              className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${viewMode === 'preview' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
              onClick={() => setViewMode('preview')}
            >
              Preview
            </button>
         </div>

         <div className="p-4 w-full min-h-100">
           {viewMode === "edit" ? (
             <Textarea 
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Start typing markdown here..."
                className="min-h-100 border-0 focus-visible:ring-0 resize-none font-mono"
             />
           ) : (
             <div className="min-h-100 prose dark:prose-invert max-w-none">
                {content.trim() ? (
                  <MarkdownPreview content={content} />
                ) : (
                  <p className="text-muted-foreground">Nothing to preview</p>
                )}
             </div>
           )}
         </div>

         <div className="bg-muted p-4 border-t w-full flex justify-end gap-2">
            <Button variant="outline" onClick={() => router.push(`/${owner}/${slug}`)}>Cancel</Button>
            <Button onClick={handleCreate} disabled={isSubmitting || !title.trim()}>
               Commit new document
            </Button>
         </div>
      </div>
    </div>
  );
}



