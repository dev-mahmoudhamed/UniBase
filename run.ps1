Write-Host "--- Installing Frontend Dependencies ---" -ForegroundColor Cyan
Set-Location frontend
npm install

if ($LASTEXITCODE -eq 0) {
    Write-Host "--- Building Angular Frontend ---" -ForegroundColor Cyan
    ng build --configuration production

    if ($LASTEXITCODE -eq 0) {
        Write-Host "--- Starting Go Backend ---" -ForegroundColor Green
        Write-Host "--- Opening Browser at http://localhost:5000 in 2 seconds... ---" -ForegroundColor Yellow
        
        # Open browser in a background job so it doesn't block the backend logs
        Start-Job -ScriptBlock {
            Start-Sleep -Seconds 2
            Start-Process "http://localhost:5000"
        } | Out-Null

        Set-Location ../backend
        go run main.go
    } else {
        Write-Host "Frontend build failed!" -ForegroundColor Red
    }
} else {
    Write-Host "Frontend dependency installation failed!" -ForegroundColor Red
}