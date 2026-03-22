
$content = Get-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Raw

$replacement = @"
  const fetchDocument = useCallback(async (resolvedId: string) => {
    try {
      const [detailRes, versionsRes] = await Promise.all([
        api.get(`/documents/`${resolvedId}`),
        api.get(`/documents/`${resolvedId}/versions`),
      ]);
"@
$content = $content -replace "const fetchDocument = useCallback\(async \(\) => \{\s*try \{\s*const \[detailRes, versionsRes\] = await Promise\.all\(\[\s*api\.get\(`/documents/\$\{itemId\}`\),\s*api\.get\(`/documents/\$\{itemId\}/versions`\),\s*\]\);", $replacement

$content = $content -replace "await fetchDocument\(\);", "await fetchDocument(resolvedItemId);"

# Need to do the same for fetchFile...

