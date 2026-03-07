Write-Host "--- Installing Frontend Dependencies ---" -ForegroundColor Cyan
Set-Location frontend
npm install

if ($LASTEXITCODE -eq 0) {
    Write-Host "--- Building Angular Frontend ---" -ForegroundColor Cyan
    ng build --configuration production

    if ($LASTEXITCODE -eq 0) {
        Write-Host "--- Starting Go Backend ---" -ForegroundColor Green
        Set-Location ../backend
        go run main.go
    } else {
        Write-Host "Frontend build failed!" -ForegroundColor Red
    }
} else {
    Write-Host "Frontend dependency installation failed!" -ForegroundColor Red
}