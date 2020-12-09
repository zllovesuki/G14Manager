$LastTag = (git describe --abbrev=0 --tags)
$ChangesCmd = "git log $LastTag..HEAD --pretty=tformat:`" (%h) - %s`""
$Changes = Invoke-Expression $ChangesCmd | Out-String

if (!$Changes) {
    Write-Host "No changes since $LastTag"
    break
}

Write-Output "Changes since $LastTag`:"
Write-Output $Changes
$NextTag = Read-Host -Prompt "Next tag"

$ChangeLog = $NextTag + "`n" + $Changes
Write-Output $ChangeLog -NoEnumerate | git tag -a -F- $NextTag
Write-Host "New tag $NextTag created."
