$env:ALIAS_SHELL = "powershell"

function al {
  if ($args[0] -in @('add', 'rm')) {
    & (Get-Command al -Type Application) @args
    if ($LASTEXITCODE -eq 0) {
      Invoke-Expression (& (Get-Command al -Type Application) init powershell | Out-String)
    }
  } else {
    & (Get-Command al -Type Application) @args
  }
}
