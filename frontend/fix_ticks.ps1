
$content = Get-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Raw

$content = $content -replace "api\.get\(/", "api.get(``/"
$content = $content -replace "}\)", "}``)"
$content = $content -replace "api\.post\(/", "api.post(``/"
$content = $content -replace "api\.patch\(/", "api.patch(``/"
$content = $content -replace "api\.delete\(/", "api.delete(``/"
$content = $content -replace "href=\{/", "href={``/"

Set-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Value $content

