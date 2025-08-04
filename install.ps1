param (
  [string]$Mode = "latest"
)

$repo = "felipebz/javm"
$os = "windows"
$arch = (Get-CimInstance Win32_Processor).Architecture

switch ($arch) {
    5 { $arch = "arm64" }
    9 { $arch = "x86_64" }
    12 { $arch = "arm64" } # Surface Pro X
    default { Write-Error "Unsupported architecture: $arch" }
}

$filename = "javm-$os-$arch.zip"
$installDir = "$env:LOCALAPPDATA\Programs\javm"

# Create safe temp dir for this run
$tempDir = Join-Path $env:TEMP ("javm-install-" + [System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
$tempFile = Join-Path $tempDir $filename

New-Item -ItemType Directory -Path $installDir -Force | Out-Null

Write-Host "Installing javm [$Mode] for $os/$arch → $installDir"

function Download-Artifact {
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        Write-Error "GitHub CLI (gh) is required for nightly install"
        exit 1
    }
    Write-Host "Downloading nightly artifact..."
    gh run download --repo $repo --name "javm-$os-$arch" --dir $tempDir
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to download nightly artifact javm-$os-$arch"
        exit 1
    }
}

function Download-Release($tag) {
    $url = "https://github.com/$repo/releases/download/$tag/$filename"
    Write-Host "Downloading $filename from release $tag..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $tempFile -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Error "Failed to download release $tag"
        exit 1
    }
}

switch ($Mode) {
    "nightly" {
        Download-Artifact
    }
    "latest" {
        $tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
        if (-not $tag) {
            Write-Error "Could not determine latest release tag"
            exit 1
        }
        Download-Release $tag
    }
    default {
        if ($Mode -match "^v?\d+(\.\d+)*$") {
            Download-Release $Mode
        } else {
            Write-Error "Usage: install.ps1 [nightly|latest|<version>]"
            exit 1
        }
    }
}

Write-Host "Extracting..."
Expand-Archive -Path $tempFile -DestinationPath $installDir -Force

Remove-Item -Recurse -Force $tempDir

# Add to PATH if missing
$envPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not ($envPath.Split(";") -contains $installDir)) {
    Write-Host "Adding $installDir to your PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$envPath;$installDir", "User")
    $env:Path += ";$installDir"
}

Write-Host "✅ javm installed successfully."
