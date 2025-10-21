function global:javm
{
    $javmExecutable = '::JAVM::'

    if ($args.Length -gt 0 -and ($args[0] -eq 'use' -or $args[0] -eq 'deactivate')) {
        $fd3 = [System.IO.Path]::GetTempFileName()
        & $javmExecutable --fd3 "$fd3" @args
        $code = $LASTEXITCODE
        Get-Content $fd3 | ForEach-Object {
            $parts = $_ -split "`t",3
            if ($parts.Length -eq 3 -and $parts[0] -eq 'SET') { Set-Item -Path env:$($parts[1]) -Value $parts[2] }
            elseif ($parts.Length -ge 2 -and $parts[0] -eq 'UNSET') { Remove-Item -ErrorAction SilentlyContinue -Path env:$($parts[1]) }
        }
        Remove-Item -Force $fd3
        $global:LASTEXITCODE = $code
    } else {
        & $javmExecutable @args
        $global:LASTEXITCODE = $LASTEXITCODE
    }
}
