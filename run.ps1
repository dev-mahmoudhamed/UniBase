Write-Host "--- Building Angular Frontend ---" -ForegroundColor Cyan
Set-Location frontend
ng build --configuration production

if ($LASTEXITCODE -eq 0) {
    Write-Host "--- Starting Go Backend ---" -ForegroundColor Green
    Set-Location ../backend
    go run main.go
} else {
    Write-Host "Frontend build failed!" -ForegroundColor Red
}