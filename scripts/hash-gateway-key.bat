@echo off
setlocal
rem Read the key inside PowerShell so cmd never expands secret input.
powershell.exe -NoLogo -NoProfile -Command "$secureKey = Read-Host 'Gateway Key (input hidden)' -AsSecureString; try { $gatewayKey = [System.Net.NetworkCredential]::new('', $secureKey).Password; if ($gatewayKey.Length -lt 43 -or $gatewayKey.Length -gt 1024 -or $gatewayKey -match '\s') { [Console]::Error.WriteLine('Key must be 43-1024 characters without whitespace.'); exit 2 }; $sha = [System.Security.Cryptography.SHA256]::Create(); try { $digest = $sha.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($gatewayKey)); [Console]::WriteLine((-join ($digest | ForEach-Object { $_.ToString('x2') }))) } finally { $sha.Dispose() } } finally { $secureKey.Dispose(); $gatewayKey = $null }"
exit /b %errorlevel%
