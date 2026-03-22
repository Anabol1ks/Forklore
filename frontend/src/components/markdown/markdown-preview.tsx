import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import remarkGfm from "remark-gfm";

type MarkdownPreviewProps = {
  content: string;
  className?: string;
};

const markdownComponents: Components = {
  h1: ({ children }) => <h1 className="text-3xl font-bold border-b pb-2 mb-4">{children}</h1>,
  h2: ({ children }) => <h2 className="text-2xl font-semibold border-b pb-2 mb-3 mt-6">{children}</h2>,
  h3: ({ children }) => <h3 className="text-xl font-semibold mb-2 mt-5">{children}</h3>,
  h4: ({ children }) => <h4 className="text-lg font-semibold mb-2 mt-4">{children}</h4>,
  p: ({ children }) => <p className="leading-7 mb-4">{children}</p>,
  a: ({ href, children }) => (
    <a href={href} target="_blank" rel="noreferrer" className="text-primary underline underline-offset-2 hover:opacity-80">
      {children}
    </a>
  ),
  ul: ({ children }) => <ul className="list-disc pl-6 mb-4 space-y-1">{children}</ul>,
  ol: ({ children }) => <ol className="list-decimal pl-6 mb-4 space-y-1">{children}</ol>,
  li: ({ children }) => <li className="leading-7">{children}</li>,
  blockquote: ({ children }) => <blockquote className="border-l-4 pl-4 text-muted-foreground my-4">{children}</blockquote>,
  hr: () => <hr className="my-6 border-border" />,
  table: ({ children }) => (
    <div className="w-full overflow-x-auto mb-4">
      <table className="w-full border-collapse text-sm">{children}</table>
    </div>
  ),
  thead: ({ children }) => <thead className="bg-muted/40">{children}</thead>,
  th: ({ children }) => <th className="border px-3 py-2 text-left font-semibold">{children}</th>,
  td: ({ children }) => <td className="border px-3 py-2 align-top">{children}</td>,
  code: ({ className, children }) => {
    const isBlock = Boolean(className);
    if (!isBlock) {
      return <code className="rounded bg-muted px-1.5 py-0.5 text-sm">{children}</code>;
    }
    return <code className="block rounded-md bg-[#0d1117] text-[#c9d1d9] p-4 text-sm overflow-x-auto">{children}</code>;
  },
  pre: ({ children }) => <pre className="mb-4">{children}</pre>,
};

export function MarkdownPreview({ content, className }: MarkdownPreviewProps) {
  return (
    <div className={className || "rounded-md border p-6 min-h-40 bg-background"}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {content || "_Пустой документ_"}
      </ReactMarkdown>
    </div>
  );
}
