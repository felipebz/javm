param (
  [string]$Mode = "latest"
)

$ErrorActionPreference = 'Stop'

$verifyAttestation = if ($env:JAVM_VERIFY_ATTESTATION) { $env:JAVM_VERIFY_ATTESTATION } else { "0" }
if ($verifyAttestation -notin @("0", "1")) {
    Write-Error "JAVM_VERIFY_ATTESTATION must be 0 or 1."
}

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
$installDir = Join-Path $env:LOCALAPPDATA "Programs/javm"

$tempRoot = Join-Path $env:TEMP ("javm-install-" + [System.Guid]::NewGuid().ToString("N"))
$tempDownloadDir = Join-Path $tempRoot "download"
$tempExtractDir = Join-Path $tempRoot "extract"

New-Item -ItemType Directory -Path $tempDownloadDir -Force | Out-Null
New-Item -ItemType Directory -Path $tempExtractDir  -Force | Out-Null
New-Item -ItemType Directory -Path $installDir -Force | Out-Null

$tempFile = Join-Path $tempDownloadDir $filename

Write-Host "Installing javm [$Mode] for $os/$arch → $installDir"

function Download-Nightly {
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        Write-Error "GitHub CLI (gh) is required for nightly install"
        exit 1
    }

    Write-Host "Downloading nightly artifact..."
    gh run download --repo $repo --name "javm-$os-$arch" --dir $tempDownloadDir | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to download nightly artifact javm-$os-$arch"
        exit 1
    }

    if (-not (Test-Path $tempFile)) {
        Write-Error "Nightly artifact not found at $tempFile"
        exit 1
    }

    $checksumPath = Get-ChildItem -Path $tempDownloadDir -File -Filter "javm_*_checksums.txt" |
        Select-Object -First 1 -ExpandProperty FullName
    if (-not $checksumPath) {
        Write-Error "Nightly checksum file not found in downloaded artifact."
        exit 1
    }

    return @{
        Tag          = "nightly"
        ZipPath      = $tempFile
        ChecksumPath = $checksumPath
    }
}

function Get-LatestTag {
    $latest = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
    return $latest.tag_name
}

function Download-Release([string]$tagName) {
    $versionNoV = $tagName.TrimStart('v')

    $releaseZipUrl = "https://github.com/$repo/releases/download/$tagName/$filename"

    Write-Host "Downloading $filename from release $tagName..."
    Invoke-WebRequest -Uri $releaseZipUrl -OutFile $tempFile -UseBasicParsing

    $checksumPath = Join-Path $tempDownloadDir "javm_$($versionNoV)_checksums.txt"
    $checksumUrl  = "https://github.com/$repo/releases/download/$tagName/javm_$($versionNoV)_checksums.txt"

    Write-Host "Downloading checksum file..."
    Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -UseBasicParsing

    return @{
        Tag          = $tagName
        ZipPath      = $tempFile
        ChecksumPath = $checksumPath
    }
}

function Verify-Checksum {
    param(
        [string]$ZipPath,
        [string]$ChecksumPath,
        [string]$ExpectedFileName
    )

    if (-not (Test-Path $ChecksumPath)) {
        Write-Error "Checksum file not found: $ChecksumPath"
        exit 1
    }

    $pattern = [regex]::Escape($ExpectedFileName) + '$'
    $expectedLine = Select-String -Path $ChecksumPath -Pattern $pattern | Select-Object -First 1
    if (-not $expectedLine) {
        Write-Error "Could not find checksum entry for $ExpectedFileName in $ChecksumPath"
        exit 1
    }

    $parts = $expectedLine.Line.Trim() -split '\s+'
    $expectedHash = $parts[0].ToLowerInvariant()

    $fileHash = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash.ToLowerInvariant()

    if ($fileHash -ne $expectedHash) {
        Write-Error "Checksum mismatch! expected $expectedHash got $fileHash"
        exit 1
    }

    Write-Host "Checksum OK."
}

function Extract-ToTemp {
    Expand-Archive -Path $tempFile -DestinationPath $tempExtractDir -Force
}

function Verify-Attestation([string]$ArtifactPath) {
    if ($verifyAttestation -ne "1") {
        Write-Host "Provenance verification skipped. Set JAVM_VERIFY_ATTESTATION=1 to verify with GitHub CLI."
        return
    }

    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
        Write-Error "GitHub CLI (gh) is required when JAVM_VERIFY_ATTESTATION=1."
    }

    if (-not (Test-Path -LiteralPath $ArtifactPath -PathType Leaf)) {
        Write-Error "Artifact not found; cannot verify attestation: $ArtifactPath"
    }

    Write-Host "Verifying attestation and provenance..."
    gh attestation verify --repo $repo $ArtifactPath | Out-Host
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Attestation verification failed."
        exit 1
    }

    Write-Host "Attestation OK."
}

function Install-To-Final {
    Write-Host "Installing to $installDir ..."
    Copy-Item -Path (Join-Path $tempExtractDir '*') -Destination $installDir -Recurse -Force
}

$result = $null

switch ($Mode) {
    "nightly" {
        $result = Download-Nightly
        Verify-Checksum -ZipPath $result.ZipPath -ChecksumPath $result.ChecksumPath -ExpectedFileName $filename
        Verify-Attestation -ArtifactPath $result.ZipPath
        Extract-ToTemp
    }

    "latest" {
        $tag = Get-LatestTag
        if (-not $tag) {
            Write-Error "Could not determine latest release tag"
            exit 1
        }

        $result = Download-Release $tag
        Verify-Checksum -ZipPath $result.ZipPath -ChecksumPath $result.ChecksumPath -ExpectedFileName $filename
        Verify-Attestation -ArtifactPath $result.ZipPath
        Extract-ToTemp
    }

    default {
        if ($Mode -match "^v?\d+(\.\d+)*$") {
            $tag = $Mode
            $result = Download-Release $tag
            Verify-Checksum -ZipPath $result.ZipPath -ChecksumPath $result.ChecksumPath -ExpectedFileName $filename
            Verify-Attestation -ArtifactPath $result.ZipPath
            Extract-ToTemp
        } else {
            Write-Error "Usage: install.ps1 [nightly|latest|<version>]"
            exit 1
        }
    }
}

Install-To-Final

Remove-Item -Recurse -Force $tempRoot

$envPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not ($envPath.Split(";") -contains $installDir)) {
    Write-Host "Adding $installDir to your PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$envPath;$installDir", "User")
    $env:Path += ";$installDir"
}

Write-Host "✅ javm installed successfully."
