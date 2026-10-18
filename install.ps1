<#
.SYNOPSIS
    Install the x-terminal release binary for this Windows machine.

.DESCRIPTION
    Works out the architecture, downloads the matching release asset and its
    published digest from GitHub, verifies the digest, and installs
    x-terminal.exe. It writes one file and needs no administrator rights.

    What the digest proves, and what it does not: the SHA256SUMS file and the
    binary come from the same release and the same host, so the digest catches a
    corrupted or truncated download. It does not prove who published it; that is
    TLS and GitHub's account, as with any release download.

.PARAMETER Prefix
    Where to install. Default: %LOCALAPPDATA%\x-terminal\bin

.PARAMETER Version
    A tag to install (for example v1.4.1). Default: the latest release.

.PARAMETER Proxy
    Proxy URL for the downloads when the system proxy is not the one to use
    (for example http://127.0.0.1:7890).

.PARAMETER AddToPath
    Append Prefix to the current user's PATH. Without it the script only prints
    the line to run; it changes no environment variable.

.EXAMPLE
    irm https://github.com/LZD121/x-terminal/releases/latest/download/install.ps1 | iex

.EXAMPLE
    irm https://github.com/LZD121/x-terminal/releases/latest/download/install.ps1 -OutFile install.ps1
    .\install.ps1 -Version v1.4.1 -AddToPath
#>
param(
    [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'x-terminal\bin'),
    [string]$Version,
    [string]$Proxy,
    [switch]$AddToPath
)

# The body is a child scope so the strictness and error preferences below belong to the install and
# are gone when it returns: `irm | iex` runs in the operator's own session.
& {
    Set-StrictMode -Version Latest
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'  # Invoke-WebRequest's progress bar is very slow

    if ($PSVersionTable.PSVersion -lt [version]'5.1') {
        throw "install.ps1: PowerShell 5.1 or later is required (this is $($PSVersionTable.PSVersion))"
    }
    # Windows PowerShell 5.1 can default to TLS 1.0, which GitHub no longer serves. Ignored where the
    # setting has no meaning (PowerShell 7 on .NET Core).
    try { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 } catch { }

    $Repo = 'LZD121/x-terminal'

    function Say([string]$Message) { Write-Host $Message }
    # throw, not exit: under `irm | iex` an exit would close the operator's shell.
    function Fail([string]$Message) { throw "install.ps1: $Message" }

    function Fetch([string]$Uri, [string]$OutFile) {
        $request = @{ Uri = $Uri; OutFile = $OutFile; UseBasicParsing = $true }
        if ($Proxy) { $request.Proxy = $Proxy }
        Invoke-WebRequest @request
    }

    # The architecture name the release assets use. RuntimeInformation answers for the OS; the
    # environment variables are the fallback for Windows PowerShell on .NET Framework before 4.7.1.
    $osArch = try { [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture } catch { $null }
    $arch = switch ("$osArch") {
        'X64' { 'amd64' }
        'AMD64' { 'amd64' }
        'Arm64' { 'arm64' }
        'ARM64' { 'arm64' }
        default { Fail "unsupported architecture $osArch (published: windows/amd64, windows/arm64)" }
    }

    if ($Version) {
        if (-not $Version.StartsWith('v')) { $Version = "v$Version" }
        $base = "https://github.com/$Repo/releases/download/$Version"
        $label = $Version
    }
    else {
        $base = "https://github.com/$Repo/releases/latest/download"
        $label = 'the latest release'
    }

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("x-terminal-install-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp -Force | Out-Null
    try {
        Say "downloading $label for windows/$arch"
        $sumsPath = Join-Path $tmp 'SHA256SUMS'
        Fetch "$base/SHA256SUMS" $sumsPath

        # The asset name carries the tag, which `latest` does not name; SHA256SUMS lists the real
        # names, so the script reads it first and takes the line for this platform.
        $suffix = "_windows_${arch}.exe"
        $asset = $null
        $expected = $null
        foreach ($line in Get-Content -LiteralPath $sumsPath) {
            $fields = -split $line
            if ($fields.Count -ge 2) {
                $name = $fields[-1].TrimStart('*')  # sha256sum's binary-mode marker
                if ($name.EndsWith($suffix)) {
                    $expected = $fields[0]
                    $asset = $name
                    break
                }
            }
        }
        if (-not $asset) { Fail "SHA256SUMS lists no asset for windows/$arch" }

        $assetPath = Join-Path $tmp $asset
        Fetch "$base/$asset" $assetPath

        Say "verifying $asset"
        $actual = (Get-FileHash -LiteralPath $assetPath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected.ToLowerInvariant()) {
            Fail "digest mismatch: published $expected, downloaded $actual"
        }
        Say 'verified against SHA256SUMS'

        New-Item -ItemType Directory -Path $Prefix -Force | Out-Null
        $target = Join-Path $Prefix 'x-terminal.exe'
        Say "installing to $target"
        Copy-Item -LiteralPath $assetPath -Destination $target -Force
        # Drop the mark-of-the-web if this PowerShell attached one, so the first run is not a
        # SmartScreen decision; there is nothing to unblock when there is no such stream.
        try { Unblock-File -LiteralPath $target -ErrorAction SilentlyContinue } catch { }

        # The binary's own report is the last check: a file that runs and names the release it came
        # from.
        $versionLine = & $target --version
        if ($LASTEXITCODE -ne 0) { Fail "$target was installed but does not run on this machine" }
        Say ($versionLine -join ' ')

        # The PATH targets are a Windows-only .NET surface; skip them anywhere else, where the
        # install itself still works and the bookkeeping has no meaning.
        if ($env:OS -eq 'Windows_NT') {
            $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
            $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
            $onPath = (($userPath, $machinePath) -join ';').Split(';', [StringSplitOptions]::RemoveEmptyEntries) |
                Where-Object { $_.TrimEnd('\') -eq $Prefix.TrimEnd('\') }
            if (-not $onPath) {
                if ($AddToPath) {
                    $newPath = if ([string]::IsNullOrEmpty($userPath)) { $Prefix } else { $userPath.TrimEnd(';') + ';' + $Prefix }
                    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
                    Say ''
                    Say "added $Prefix to your user PATH; open a new terminal for it to take effect"
                }
                else {
                    Say ''
                    Say "$Prefix is not on PATH. Add it for this session:"
                    Say "  `$env:Path = `"$Prefix;`$env:Path`""
                    Say 'or permanently, by re-running this script with -AddToPath'
                }
            }
        }

        Say ''
        Say 'next: x-terminal login   (then x-terminal doctor)'
    }
    finally {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}
