$env:ALIAS_SHELL = "powershell"

function global:al {
  if ($args[0] -in @('add', 'rm', 'edit')) {
    & (Get-Command al -Type Application) @args
    if ($LASTEXITCODE -eq 0) {
      if ($args[0] -ne 'add') {
        Remove-Item "Function:$($args[1])" -ErrorAction SilentlyContinue
      }
      Invoke-Expression (& (Get-Command al -Type Application) init powershell | Out-String)
    }
  } else {
    & (Get-Command al -Type Application) @args
  }
}
