# Adds Windows Firewall rules so chinook e2e test binaries can listen on
# 0.0.0.0 (needed by the in-process mock OpenAI server reachable from Docker)
# without triggering the "Allow this app to communicate" popup on every run.
#
# Run once, as Administrator:
#     powershell.exe -ExecutionPolicy Bypass -File scripts\setup-windows-firewall.ps1
#
# Re-running is safe: existing rules with the same DisplayName are removed
# first so the program path stays in sync if you move the repo.

#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$binDir = Join-Path $repoRoot "bin\chinooktest"

$paths = @(
    @{ Name = "ChinookE2E - test binary";     Path = Join-Path $binDir "chinooktest.test.exe" },
    @{ Name = "ChinookE2E - provider binary"; Path = Join-Path $binDir "terraform-provider-typesense.exe" }
)

foreach ($entry in $paths) {
    $name = $entry.Name
    $path = $entry.Path

    Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Remove-NetFirewallRule

    if (-not (Test-Path $path)) {
        Write-Host "Skipping $name (build it first with 'make chinook-e2e'): $path" -ForegroundColor Yellow
        continue
    }

    New-NetFirewallRule `
        -DisplayName $name `
        -Direction Inbound `
        -Action Allow `
        -Program $path `
        -Profile Any `
        -Protocol TCP `
        | Out-Null

    Write-Host "Allowed: $name -> $path" -ForegroundColor Green
}

Write-Host ""
Write-Host "Done. The chinook e2e test binary can now listen without a firewall prompt."
