$global:_javmExecutable = '::JAVM::'

function javm
{
    $fd3=$([System.IO.Path]::GetTempFileName())
    & $global:_javmExecutable $args --fd3 "$fd3"
    $fd3content=$(Get-Content $fd3)
    if ($fd3content) {
        $expression=$fd3content.replace("export ","`$env:").replace("unset ","Remove-Item env:") -join "`n"
        if (-not $expression -eq "") { Invoke-Expression $expression }
    }
    Remove-Item -Force $fd3
}
