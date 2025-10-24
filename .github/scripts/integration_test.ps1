#!/usr/bin/env pwsh
# Intended to be dot-sourced after initializing: iex "$(javm init pwsh)"
# Example:
#   $ErrorActionPreference = 'Stop'
#   Invoke-Expression (javm init pwsh)
#   . .github/scripts/integration_test.ps1
#
# Optional: set $env:JAVA_VERSION = "21"

$ErrorActionPreference = 'Stop'

# Resolve Java version
$javaVersion = if ($env:JAVA_VERSION -and $env:JAVA_VERSION -ne '') { $env:JAVA_VERSION } else { '25' }

Write-Host '>>> Exercising discovery/ls-remote'
javm ls-remote 21
javm discover list

Write-Host ">>> Installing JDK $javaVersion (idempotent)"
javm install "$javaVersion"

Write-Host ">>> Using JDK $javaVersion"
javm use "$javaVersion"

Write-Host '>>> Validations'
if (-not $env:JAVA_HOME -or $env:JAVA_HOME -eq '') { Write-Error 'JAVA_HOME is empty'; exit 1 }
if (-not (Test-Path (Join-Path $env:JAVA_HOME 'bin/java.exe'))) { Write-Error "java.exe not found in $env:JAVA_HOME/bin"; exit 1 }

# java resolution on PATH
$javaCmd = Get-Command java -ErrorAction Stop
$javaPath = $javaCmd.Source
Write-Host "java on PATH: $javaPath"
java --version

# Internal mapping
$expectHome = javm which "$javaVersion" --home
if ($env:JAVA_HOME -ne $expectHome) { Write-Error "JAVA_HOME ($env:JAVA_HOME) != javm which ($expectHome)"; exit 1 }

# Ensure java on PATH points to current JAVA_HOME
$javaPathNorm = [System.IO.Path]::GetFullPath($javaPath)
$javaHomeNorm = [System.IO.Path]::GetFullPath($env:JAVA_HOME)

if ($javaPathNorm.StartsWith($javaHomeNorm, [StringComparison]::OrdinalIgnoreCase)) {
  Write-Host 'OK: java on PATH matches JAVA_HOME'
} else {
  Write-Error "java on PATH ($javaPathNorm) does not match JAVA_HOME ($javaHomeNorm)"
  exit 1
}

Write-Host '>>> Done.'
