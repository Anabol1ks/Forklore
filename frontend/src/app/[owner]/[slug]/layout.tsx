import { RepoProvider } from "@/components/repo/repo-context";
import { RepoLayoutComponent } from "@/components/repo/repo-layout";
import { ReactNode } from "react";

export default async function RepoLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ owner: string; slug: string }>;
}) {
  const resolvedParams = await params;
  
  return (
    <RepoProvider owner={resolvedParams.owner} slug={resolvedParams.slug}>
      <RepoLayoutComponent owner={resolvedParams.owner} slug={resolvedParams.slug}>
        {children}
      </RepoLayoutComponent>
    </RepoProvider>
  );
}
