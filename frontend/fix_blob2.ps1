
$content = Get-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Raw

$replacement1 = @"
  const [resolvedTargetId, setResolvedTargetId] = useState<string>("");

  const fetchRepo = useCallback(async (): Promise<{ repo: Repository; repoId: string }> => {
    const bySlug = await api.get(`/users/`${owner}`/repositories/`${repoSlug}`);
    const slugRepo = (bySlug.data.repository || bySlug.data) as Repository;
    const repoId = getId(slugRepo as unknown as Record<string, unknown>, ["id", "repo_id"]);
    if (!repoId) throw new Error("Repository id is missing");

    const byID = await api.get(`/repositories/`${repoId}`);
    const repoData = (byID.data.repository || slugRepo) as Repository;
    setRepo(repoData);
    return { repo: repoData, repoId };
  }, [owner, repoSlug]);

  const fetchDocument = useCallback(async (targetId: string) => {
    const [detailRes, versionsRes] = await Promise.all([
      api.get(`/documents/`${targetId}`),
      api.get(`/documents/`${targetId}`/versions`),
    ]);
"@
$content = $content -replace 'const fetchRepo = useCallback\(async \(\): Promise<Repository> => \{[\s\S]*?const fetchDocument = useCallback\(async \(\) => \{\s*const \[detailRes, versionsRes\] = await Promise\.all\(\[\s*api\.get\(`/documents/\$\{itemId\}`\),\s*api\.get\(`/documents/\$\{itemId\}/versions`\),\s*\]\);', $replacement1

$replacement2 = @"
    setSelectedDocumentVersionSummary("");
    setDocumentViewMode("preview");
  }, []);

  const fetchFile = useCallback(async (targetId: string) => {
    const [detailRes, versionsRes] = await Promise.all([
      api.get(`/files/`${targetId}`),
      api.get(`/files/`${targetId}`/versions`),
    ]);
"@
$content = $content -replace 'setSelectedDocumentVersionSummary\(""\);\s*setDocumentViewMode\("preview"\);\s*\}, \[itemId\]\);\s*const fetchFile = useCallback\(async \(\) => \{\s*const \[detailRes, versionsRes\] = await Promise\.all\(\[\s*api\.get\(`/files/\$\{itemId\}`\),\s*api\.get\(`/files/\$\{itemId\}/versions`\),\s*\]\);', $replacement2

$content = $content -replace '\}, \[itemId\]\);', '}, []);'

$replacementEffect = @"
  useEffect(() => {
    const run = async () => {
      try {
        setLoading(true);
        const { repoId } = await fetchRepo();
        
        let targetId = itemId;
        
        // Find correct ID if itemId is actually a name/slug
        if (isDocument) {
           const docsRes = await api.get(`/repositories/`${repoId}`/documents`);
           const docs = (docsRes.data.documents || []) as any[];
           const decodedItemId = decodeURIComponent(itemId);
           const found = docs.find(d => d.id === decodedItemId || d.document_id === decodedItemId || d.slug === decodedItemId || d.title === decodedItemId);
           if (found) {
               targetId = found.id || found.document_id;
           }
           setResolvedTargetId(targetId);
           await fetchDocument(targetId);
        } else if (isFile) {
           const filesRes = await api.get(`/repositories/`${repoId}`/files`);
           const files = (filesRes.data.files || []) as any[];
           const decodedItemId = decodeURIComponent(itemId);
           const found = files.find(f => f.id === decodedItemId || f.file_id === decodedItemId || f.file_name === decodedItemId);
           if (found) {
               targetId = found.id || found.file_id;
           }
           setResolvedTargetId(targetId);
           await fetchFile(targetId);
        } else {
          throw new Error("Unsupported blob type");
        }
      } catch (error) {
"@
$content = $content -replace 'useEffect\(\(\) => \{\s*const run = async \(\) => \{\s*try \{\s*setLoading\(true\);\s*await fetchRepo\(\);\s*if \(isDocument\) \{\s*await fetchDocument\(\);\s*\} else if \(isFile\) \{\s*await fetchFile\(\);\s*\} else \{\s*throw new Error\("Unsupported blob type"\);\s*\}\s*\} catch \(error\) \{', $replacementEffect

$content = $content -replace 'await fetchDocument\(\);', 'await fetchDocument(resolvedTargetId);'
$content = $content -replace 'await fetchFile\(\);', 'await fetchFile(resolvedTargetId);'

$replacementUrl = @"
        const params = activeFileVersionId ? { version_id: activeFileVersionId } : undefined;
        const response = await api.get(`/files/`${resolvedTargetId || itemId}`/content`, {
"@
$content = $content -replace 'const params = activeFileVersionId \? \{ version_id: activeFileVersionId \} : undefined;\s*const response = await api\.get\(`/files/\$\{itemId\}/content`, \{', $replacementUrl

$content = $content -replace '/documents/\$\{itemId\}', '/documents/${resolvedTargetId || itemId}'
$content = $content -replace '/files/\$\{itemId\}', '/files/${resolvedTargetId || itemId}'

Set-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Value $content

