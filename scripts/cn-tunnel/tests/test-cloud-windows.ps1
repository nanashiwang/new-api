param([string]$Installer)
$ErrorActionPreference = 'Stop'
. $Installer
$Work = Join-Path $env:TEMP ([Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $Work | Out-Null
try {
    $Key = [Convert]::ToBase64String((New-Object byte[] 32))
    $Config = "[Interface]`nPrivateKey = $Key`nAddress = 10.66.0.2/32`nMTU = 1280`n`n[Peer]`nPublicKey = $Key`nPresharedKey = $Key`nEndpoint = 192.0.2.1:51820`nAllowedIPs = 10.66.0.1/32`nPersistentKeepalive = 25`n"
    $Version = (Get-Content (Join-Path $PSScriptRoot '../VERSION') -Raw).Trim()
    $Payload = @{ version=$Version; name='win-cloud'; domain='cn.meta-api.vip'; ip='10.66.0.1'; config=$Config }
    $Code = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(($Payload | ConvertTo-Json -Compress)))
    $Name = New-CnClientFiles $Code $Work
    if ($Name -ne 'win-cloud') { throw 'Wrong device name' }
    if ([IO.File]::ReadAllText((Join-Path $Work 'win-cloud.conf')) -cne $Config) { throw 'Configuration mismatch' }
    Get-ChildItem $Work -Filter '*.ps1' | ForEach-Object {
        $Text = [IO.File]::ReadAllText($_.FullName)
        if ($Text -match '__TUNNEL_|__CONFIG_|__SERVICE_') { throw 'Unrendered placeholders' }
        $Tokens = $null; $ParseErrors = $null
        [System.Management.Automation.Language.Parser]::ParseFile($_.FullName, [ref]$Tokens, [ref]$ParseErrors) | Out-Null
        if ($ParseErrors.Count -gt 0) { throw ($ParseErrors | Out-String) }
        $Bytes = [IO.File]::ReadAllBytes($_.FullName)
        if ($Bytes[0] -ne 239 -or $Bytes[1] -ne 187 -or $Bytes[2] -ne 191) { throw 'Missing UTF8 BOM' }
    }
    foreach ($BadConfig in @($Config + "PostUp = echo unsafe`n", $Config.Replace('10.66.0.1/32', '0.0.0.0/0'))) {
        $Payload.config = $BadConfig
        $BadCode = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes(($Payload | ConvertTo-Json -Compress)))
        $Rejected = $false
        try { New-CnClientFiles $BadCode $Work | Out-Null } catch { $Rejected = $true }
        if (-not $Rejected) { throw 'Unsafe configuration accepted' }
    }
    Write-Host 'Windows cloud provisioning tests passed'
} finally {
    Remove-Item -LiteralPath $Work -Recurse -Force
}
