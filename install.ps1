param (
  [string]$Mode = "latest",
  [string]$Version = ""
)

$repo = "felipebz/javm"
$os = "windows"
$arch = (Get-CimInstance Win32_Processor).Architecture
$installDir = "$env:LOCALAPPDATA\Programs\javm"

switch ($arch) {
    5 { $arch = "arm64" }
    9 { $arch = "x86_64" }
    12 { $arch = "arm64" } # Surface Pro X
    default { Write-Error "Unsupported architecture: $arch" }
}

$filename = "javm-$os-$arch.zip"
$destFile = "$installDir\$filename"

Write-Host "Installing javm [$Mode] for $os/$arch..."

New-Item -Path $installDir -ItemType Directory -Force | Out-Null

function Download-Artifact {
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        Write-Error "GitHub CLI (gh) is required for nightly install"
        exit 1
    }
    Write-Host "Downloading latest nightly artifact..."
    gh run download --repo $repo --name "javm-$os-$arch" --dir $installDir
}

function Download-Release($tag) {
    $url = "https://github.com/$repo/releases/download/$tag/$filename"
    Write-Host "Downloading $filename from release $tag..."
    Invoke-WebRequest -Uri $url -OutFile $destFile -UseBasicParsing
}

function Extract {
    Write-Host "Extracting..."
    Expand-Archive -Path $destFile -DestinationPath $installDir -Force
    Remove-Item -Force $destFile
    Write-Host "✅ Installed to $installDir"
}

switch ($Mode) {
    "nightly" {
        Download-Artifact
    }
    "latest" {
        $tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
        Download-Release $tag
    }
    default {
        if ($Mode -match "^v?\d+(\.\d+)*$") {
            Download-Release $Mode
        } else {
            Write-Error "Usage: .\install.ps1 [nightly|latest|<version>]"
            exit 1
        }
    }
}

Extract
