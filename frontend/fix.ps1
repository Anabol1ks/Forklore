
$content = Get-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Raw
$content = $content -replace '<span className="font-medium text-lg">\{isDocument \? String\(document\?\.title \|\| "Документ"\) : String\(file\?\.file_name \|\| "Файл"\)\}</span>\s*\{isDocument && document \? \(', '<span className="font-medium text-lg">{isDocument ? String(document?.title || "Документ") : String(file?.file_name || "Файл")}</span>
        </div>

      {isDocument && document ? ('
Set-Content -LiteralPath "c:\Users\Grigo\Documents\GitGrisha\Forklore\frontend\src\app\[owner]\[slug]\blob\[type]\[itemId]\page.tsx" -Value $content

