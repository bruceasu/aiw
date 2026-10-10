param(
    [Parameter(Mandatory = $true)][string]$Executable,
    [Parameter(Mandatory = $true)][ValidateSet('ja', 'en')][string]$Target,
    [ValidateSet('ja-business')][string]$Profile = ''
)

$ErrorActionPreference = 'Stop'

try {
    $encodedInput = [Console]::In.ReadLine()
    if ($null -eq $encodedInput) { throw 'No clipboard text was received.' }
    $text = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($encodedInput))

    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Executable
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.RedirectStandardInput = $true
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $utf8 = [Text.UTF8Encoding]::new($false)
    $start.StandardInputEncoding = $utf8
    $start.StandardOutputEncoding = $utf8
    $start.StandardErrorEncoding = $utf8
    $start.Arguments = "--target $Target"
    if ($Profile -ne '') { $start.Arguments += " --profile $Profile" }

    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $start
    if (-not $process.Start()) { throw 'Could not start aiw-say.exe.' }
    $process.StandardInput.Write($text)
    $process.StandardInput.Close()
    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()
    $process.WaitForExit()
    $stdout = $stdoutTask.Result
    $stderr = $stderrTask.Result

    $stdout64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($stdout))
    $stderr64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($stderr))
    if ($stdout64 -eq '') { $stdout64 = '-' }
    if ($stderr64 -eq '') { $stderr64 = '-' }
    [Console]::Write("{0}|{1}|{2}", $process.ExitCode, $stdout64, $stderr64)
} catch {
    [Console]::Error.Write($_.Exception.Message)
    exit 1
}
