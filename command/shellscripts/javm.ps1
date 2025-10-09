function global:javm
{
    $javmExecutable = '::JAVM::'
    $fd3 = [System.IO.Path]::GetTempFileName()
    & $javmExecutable --fd3 "$fd3" @args
    $fd3content = Get-Content $fd3
    if ($fd3content)
    {
        $expression = $fd3content.replace("export ", "`$env:").replace("unset ", "Remove-Item env:") -join "`n"
        if ($expression -ne "")
        {
            Invoke-Expression $expression
        }
    }
    Remove-Item -Force $fd3
}
