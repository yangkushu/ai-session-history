param(
    [string]$Version = $env:AI_HISTORY_VERSION,
    [string]$InstallDir = $env:AI_HISTORY_INSTALL_DIR,
    [switch]$NoModifyPath,
    [switch]$WithSkill,
    [ValidateScript({
        foreach ($Name in ($_ -split ',')) {
            if ($Name -notin @('codex', 'claude-code', 'cursor')) {
                throw "unsupported Agent '$Name' (supported: codex, claude-code, cursor)"
            }
        }
        return $true
    })]
    [string[]]$Agent
)

$ErrorActionPreference = 'Stop'
$Repository = 'yangkushu/ai-session-history'

function Get-BinaryVersion {
    param([Parameter(Mandatory = $true)][string]$Path)

    try {
        $OutputLines = @(& $Path version 2>$null)
        if ($LASTEXITCODE -ne 0 -or $OutputLines.Count -eq 0) {
            return $null
        }
        $FirstLine = [string]$OutputLines[0]
        if ($FirstLine.Trim() -match '^ai-history (v[0-9]+\.[0-9]+\.[0-9]+)$') {
            return $Matches[1]
        }
    }
    catch {
        return $null
    }
    return $null
}

function Get-NormalizedPathEntry {
    param([Parameter(Mandatory = $true)][string]$Path)

    $Value = $Path.Trim().Trim('"')
    try {
        return [System.IO.Path]::GetFullPath($Value).TrimEnd('\', '/')
    }
    catch {
        return $Value.TrimEnd('\', '/')
    }
}

function Get-UserPath {
    if ($env:AI_HISTORY_TEST_USER_PATH_FILE) {
        if (Test-Path -LiteralPath $env:AI_HISTORY_TEST_USER_PATH_FILE -PathType Leaf) {
            return [System.IO.File]::ReadAllText($env:AI_HISTORY_TEST_USER_PATH_FILE)
        }
        return ''
    }
    $Value = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($null -eq $Value) { return '' }
    return $Value
}

function Set-UserPath {
    param([Parameter(Mandatory = $true)][AllowEmptyString()][string]$Value)

    if ($env:AI_HISTORY_TEST_USER_PATH_FILE) {
        $Parent = Split-Path -Parent $env:AI_HISTORY_TEST_USER_PATH_FILE
        if ($Parent) { [System.IO.Directory]::CreateDirectory($Parent) | Out-Null }
        [System.IO.File]::WriteAllText($env:AI_HISTORY_TEST_USER_PATH_FILE, $Value)
        return
    }
    [Environment]::SetEnvironmentVariable('Path', $Value, 'User')
}

function Add-InstallDirToUserPath {
    param([Parameter(Mandatory = $true)][string]$Directory)

    $Wanted = Get-NormalizedPathEntry $Directory
    $Entries = New-Object System.Collections.Generic.List[string]
    $Found = $false
    foreach ($Entry in ((Get-UserPath) -split ';')) {
        if ([string]::IsNullOrWhiteSpace($Entry)) { continue }
        if ([string]::Equals((Get-NormalizedPathEntry $Entry), $Wanted, [StringComparison]::OrdinalIgnoreCase)) {
            if (-not $Found) {
                $Entries.Add($Directory)
                $Found = $true
            }
            continue
        }
        $Entries.Add($Entry)
    }
    if (-not $Found) { $Entries.Add($Directory) }
    Set-UserPath ($Entries -join ';')
}

function Get-SelectedAgents {
    param([string[]]$ExplicitAgents, [string]$HomeDirectory)

    $Selected = New-Object System.Collections.Generic.List[string]
    foreach ($Entry in @($ExplicitAgents)) {
        foreach ($Name in ($Entry -split ',')) {
            if ([string]::IsNullOrWhiteSpace($Name)) { continue }
            $Normalized = $Name.Trim().ToLowerInvariant()
            if ($Normalized -notin @('codex', 'claude-code', 'cursor')) {
                throw "unsupported Agent '$Name' (supported: codex, claude-code, cursor)"
            }
            $Selected.Add($Normalized)
        }
    }
    if ($Selected.Count -gt 0) { return @($Selected) }

    if ((Get-Command codex -ErrorAction SilentlyContinue) -or
        (Test-Path -LiteralPath (Join-Path $HomeDirectory '.codex') -PathType Container)) {
        $Selected.Add('codex')
    }
    if ((Get-Command claude -ErrorAction SilentlyContinue) -or
        (Test-Path -LiteralPath (Join-Path $HomeDirectory '.claude') -PathType Container)) {
        $Selected.Add('claude-code')
    }
    if ((Get-Command cursor -ErrorAction SilentlyContinue) -or
        (Test-Path -LiteralPath (Join-Path $HomeDirectory '.cursor') -PathType Container)) {
        $Selected.Add('cursor')
    }
    return @($Selected)
}

function Invoke-AiHistoryInstaller {
    if (-not $script:InstallDir) {
        if (-not $env:LOCALAPPDATA) { throw 'LOCALAPPDATA is unavailable; specify -InstallDir' }
        $script:InstallDir = Join-Path $env:LOCALAPPDATA 'ai-history\bin'
    }
    $script:InstallDir = [System.IO.Path]::GetFullPath($script:InstallDir)

    $ReleaseBaseUrl = if ($env:AI_HISTORY_RELEASE_BASE_URL) {
        $env:AI_HISTORY_RELEASE_BASE_URL.TrimEnd('/')
    }
    else {
        "https://github.com/$Repository/releases/download"
    }
    $LatestReleaseUrl = if ($env:AI_HISTORY_LATEST_RELEASE_URL) {
        $env:AI_HISTORY_LATEST_RELEASE_URL
    }
    else {
        "https://api.github.com/repos/$Repository/releases/latest"
    }

    $Target = Join-Path $script:InstallDir 'ai-history.exe'
    $ExistingVersion = $null
    if (Test-Path -LiteralPath $Target) {
        if (-not (Test-Path -LiteralPath $Target -PathType Leaf)) {
            throw "existing target is not a file: $Target"
        }
        $ExistingVersion = Get-BinaryVersion $Target
        if (-not $ExistingVersion) {
            throw "existing target is not a recognized ai-history binary: $Target"
        }
    }

    $RawOS = if ($env:AI_HISTORY_TEST_OS) { $env:AI_HISTORY_TEST_OS } else { 'windows' }
    if ($RawOS.ToLowerInvariant() -ne 'windows') {
        throw "unsupported operating system: $RawOS (supported: windows)"
    }
    $RawArchitecture = if ($env:AI_HISTORY_TEST_ARCH) {
        $env:AI_HISTORY_TEST_ARCH
    }
    else {
        $env:PROCESSOR_ARCHITECTURE
    }
    switch ($RawArchitecture.ToLowerInvariant()) {
        { $_ -in @('amd64', 'x86_64') } { $Architecture = 'amd64'; break }
        { $_ -in @('arm64', 'aarch64') } { $Architecture = 'arm64'; break }
        default { throw "unsupported architecture: $RawArchitecture (supported: amd64, arm64)" }
    }

    if (-not $script:Version) {
        try {
            $LatestResponse = Invoke-WebRequest -UseBasicParsing -Uri $LatestReleaseUrl
            $LatestDocument = $LatestResponse.Content | ConvertFrom-Json
            $script:Version = [string]$LatestDocument.tag_name
        }
        catch {
            throw "could not resolve latest release from $LatestReleaseUrl; specify -Version vX.Y.Z: $($_.Exception.Message)"
        }
        if (-not $script:Version) {
            throw "latest release response from $LatestReleaseUrl has no tag_name; specify -Version vX.Y.Z"
        }
    }
    if ($script:Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') {
        throw "invalid version '$script:Version'; expected vX.Y.Z"
    }

    $SkipBinary = $ExistingVersion -eq $script:Version
    if ($SkipBinary) {
        Write-Output "ai-history $($script:Version) is already installed at $Target"
    }
    else {
        $ArchiveVersion = $script:Version.TrimStart('v')
        $ArchiveName = "ai-history_${ArchiveVersion}_windows_${Architecture}.zip"
        $ReleaseUrl = "$ReleaseBaseUrl/$($script:Version)"
        $TempDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("ai-history-install-" + [Guid]::NewGuid().ToString('N'))
        $ArchivePath = Join-Path $TempDirectory $ArchiveName
        $ChecksumsPath = Join-Path $TempDirectory 'checksums.txt'
        $StageDirectory = Join-Path $TempDirectory 'stage'
        $NewTarget = Join-Path $script:InstallDir ".ai-history.new.$PID.exe"
        $BackupTarget = Join-Path $script:InstallDir ".ai-history.backup.$PID.exe"
        $NewTargetCreated = $false
        $BackupCreated = $false
        $ReplacementCommitted = $false

        try {
            [System.IO.Directory]::CreateDirectory($TempDirectory) | Out-Null
            try {
                Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseUrl/checksums.txt" -OutFile $ChecksumsPath
                Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseUrl/$ArchiveName" -OutFile $ArchivePath
            }
            catch {
                throw "download failed for release $($script:Version) from $ReleaseUrl`: $($_.Exception.Message)"
            }

            $ChecksumMatches = @()
            foreach ($Line in (Get-Content -LiteralPath $ChecksumsPath)) {
                if ($Line -match '^([0-9A-Fa-f]{64})\s{2,}(.+?)\s*$' -and $Matches[2] -eq $ArchiveName) {
                    $ChecksumMatches += $Matches[1]
                }
            }
            if ($ChecksumMatches.Count -ne 1) {
                throw "checksum entry for $ArchiveName must appear exactly once"
            }
            $ActualChecksum = (Get-FileHash -LiteralPath $ArchivePath -Algorithm SHA256).Hash
            if (-not [string]::Equals($ChecksumMatches[0], $ActualChecksum, [StringComparison]::OrdinalIgnoreCase)) {
                throw "checksum verification failed for $ArchiveName"
            }

            [System.IO.Directory]::CreateDirectory($StageDirectory) | Out-Null
            Expand-Archive -LiteralPath $ArchivePath -DestinationPath $StageDirectory -Force
            $StagedBinary = Join-Path $StageDirectory 'ai-history.exe'
            if (-not (Test-Path -LiteralPath $StagedBinary -PathType Leaf)) {
                throw "downloaded archive has no ai-history.exe"
            }
            $StagedVersion = Get-BinaryVersion $StagedBinary
            if ($StagedVersion -ne $script:Version) {
                throw "downloaded binary version does not match $($script:Version)"
            }

            [System.IO.Directory]::CreateDirectory($script:InstallDir) | Out-Null
            if (Test-Path -LiteralPath $NewTarget) {
                throw "staging target already exists: $NewTarget"
            }
            if (Test-Path -LiteralPath $BackupTarget) {
                throw "backup target already exists: $BackupTarget"
            }
            [System.IO.File]::Copy($StagedBinary, $NewTarget, $false)
            $NewTargetCreated = $true
            $HadExisting = Test-Path -LiteralPath $Target -PathType Leaf
            try {
                if ($HadExisting) {
                    Move-Item -LiteralPath $Target -Destination $BackupTarget
                    $BackupCreated = $true
                }
                Move-Item -LiteralPath $NewTarget -Destination $Target
                $NewTargetCreated = $false
                $InstalledVersion = Get-BinaryVersion $Target
                if ($InstalledVersion -ne $script:Version) {
                    throw "installed binary version does not match $($script:Version)"
                }
                $ReplacementCommitted = $true
                if (Test-Path -LiteralPath $BackupTarget) {
                    Remove-Item -LiteralPath $BackupTarget -Force
                }
            }
            catch {
                $ReplacementError = $_.Exception.Message
                if ($BackupCreated -and (Test-Path -LiteralPath $BackupTarget -PathType Leaf)) {
                    if (Test-Path -LiteralPath $Target) { Remove-Item -LiteralPath $Target -Force }
                    try {
                        Move-Item -LiteralPath $BackupTarget -Destination $Target
                        $BackupCreated = $false
                    }
                    catch {
                        throw "installation failed and backup restore failed; backup retained at $BackupTarget`: $ReplacementError; $($_.Exception.Message)"
                    }
                }
                elseif (Test-Path -LiteralPath $Target) {
                    Remove-Item -LiteralPath $Target -Force
                }
                throw "installation failed; previous binary restored: $ReplacementError"
            }
            Write-Output "Installed ai-history $($script:Version) at $Target"
        }
        finally {
            if (Test-Path -LiteralPath $TempDirectory) { Remove-Item -LiteralPath $TempDirectory -Recurse -Force -ErrorAction SilentlyContinue }
            if ($NewTargetCreated -and (Test-Path -LiteralPath $NewTarget)) {
                Remove-Item -LiteralPath $NewTarget -Force -ErrorAction SilentlyContinue
            }
            if ($ReplacementCommitted -and $BackupCreated -and (Test-Path -LiteralPath $BackupTarget)) {
                Remove-Item -LiteralPath $BackupTarget -Force
            }
        }
    }

    if (-not $NoModifyPath) {
        Add-InstallDirToUserPath $script:InstallDir
    }

    $VersionAfterInstall = Get-BinaryVersion $Target
    if ($VersionAfterInstall -ne $script:Version) {
        throw "installed binary identity check failed at $Target"
    }

    try {
        $DoctorLines = @(& $Target doctor --json 2>$null)
        if ($LASTEXITCODE -ne 0) { throw 'doctor command returned nonzero' }
        $DoctorText = ($DoctorLines -join "`n").Trim()
        if (-not ($DoctorText.StartsWith('[') -and $DoctorText.EndsWith(']'))) {
            throw 'doctor output is not a JSON array'
        }
        $null = $DoctorText | ConvertFrom-Json
    }
    catch {
        throw "ai-history doctor --json failed or returned invalid JSON diagnostics: $($_.Exception.Message)"
    }

    $PathCommand = Get-Command ai-history -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($PathCommand) {
        $PathBinary = if ($PathCommand.Path) { $PathCommand.Path } else { $PathCommand.Source }
        if ($PathBinary -and -not [string]::Equals(
            (Get-NormalizedPathEntry $PathBinary),
            (Get-NormalizedPathEntry $Target),
            [StringComparison]::OrdinalIgnoreCase)) {
            Write-Warning "PATH resolves ai-history to $PathBinary instead of installed $Target"
        }
    }

    if ($WithSkill) {
        $HomeDirectory = if ($env:USERPROFILE) { $env:USERPROFILE } elseif ($env:HOME) { $env:HOME } else { '' }
        if (-not $HomeDirectory) { throw 'user home is unavailable for Agent detection' }
        $SelectedAgents = @(Get-SelectedAgents -ExplicitAgents $Agent -HomeDirectory $HomeDirectory)
        if ($SelectedAgents.Count -eq 0) {
            throw 'no supported Agent detected; specify -Agent codex, -Agent claude-code, or -Agent cursor'
        }

        $NpxCommand = Get-Command npx -ErrorAction SilentlyContinue | Select-Object -First 1
        if (-not $NpxCommand) { throw 'required tool is unavailable for Skill installation: npx' }
        $NpxPath = if ($NpxCommand.Path) { $NpxCommand.Path } else { $NpxCommand.Source }
        $SkillSource = "https://github.com/$Repository/tree/$($script:Version)/skills/ai-history"
        $FailedAgents = New-Object System.Collections.Generic.List[string]
        foreach ($Name in $SelectedAgents) {
            Write-Output "Installing ai-history Skill for $Name"
            & $NpxPath --yes skills add $SkillSource --skill ai-history --global --agent $Name --yes
            if ($LASTEXITCODE -eq 0) {
                Write-Output "Installed ai-history Skill for $Name"
            }
            else {
                Write-Error "Failed to install ai-history Skill for $Name" -ErrorAction Continue
                $FailedAgents.Add($Name)
            }
        }
        if ($FailedAgents.Count -gt 0) {
            throw "partial Skill installation failure; failed Agents: $($FailedAgents -join ', ')"
        }
    }
}

try {
    Invoke-AiHistoryInstaller
}
catch {
    Write-Error $_.Exception.Message
    exit 1
}
