$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$Repo = 'nqode-io/qode'
$InstallDir = Join-Path $env:LOCALAPPDATA 'qode\bin'
$BinaryPath = Join-Path $InstallDir 'qode.exe'

if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') {
  Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
  exit 1
}
$Arch = 'amd64'

# Beta-channel installer: matches any v* tag, including -beta pre-releases.
# Post-beta switch to /releases/latest is documented in docs/versioning.md.
$Releases = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases?per_page=30" `
  -Headers @{ 'Accept' = 'application/vnd.github.v3+json' }
$Release = $Releases | Where-Object { $_.tag_name -match '^v\d' } | Select-Object -First 1
if (-not $Release) { Write-Error "No tagged v* release found in $Repo"; exit 1 }
$Version = $Release.tag_name

$Filename     = "qode_${Version}_windows_${Arch}.zip"
$Base         = "https://github.com/$Repo/releases/download/$Version"
$DownloadUrl  = "$Base/$Filename"
$ChecksumsUrl = "$Base/checksums.txt"

$TmpDir = Join-Path $env:TEMP ("qode-install-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $TmpDir | Out-Null
try {
  $ZipPath       = Join-Path $TmpDir $Filename
  $ChecksumsPath = Join-Path $TmpDir 'checksums.txt'
  Invoke-WebRequest $DownloadUrl  -OutFile $ZipPath
  Invoke-WebRequest $ChecksumsUrl -OutFile $ChecksumsPath

  $Pattern = "^[0-9a-fA-F]{64}\s+$([regex]::Escape($Filename))$"
  $Line = Get-Content $ChecksumsPath | Where-Object { $_ -match $Pattern }
  if (-not $Line) { Write-Error "checksum entry missing or malformed for $Filename"; exit 1 }
  $Expected = ($Line -split '\s+' | Select-Object -First 1).ToLower()
  $Actual   = (Get-FileHash $ZipPath -Algorithm SHA256).Hash.ToLower()
  if ($Actual -ne $Expected) { Write-Error "Checksum mismatch. Aborting."; exit 1 }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Expand-Archive $ZipPath -DestinationPath $TmpDir -Force
  Move-Item (Join-Path $TmpDir 'qode.exe') $BinaryPath -Force

  $UserPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
  $Parts = @(); if ($UserPath) { $Parts = $UserPath -split ';' | Where-Object { $_ } }
  if ($Parts -notcontains $InstallDir) {
    $NewPath = (($Parts + $InstallDir) -join ';')
    [Environment]::SetEnvironmentVariable('PATH', $NewPath, 'User')
    Write-Host "Added $InstallDir to user PATH. Restart your terminal."
  }
  Write-Host "qode $Version installed to $BinaryPath"
  Write-Host "Note: Windows SmartScreen may block first run. Click 'More info' -> 'Run anyway'."
} finally {
  Remove-Item $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
