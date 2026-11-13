#!/bin/bash
set -e # Exit immediately if a command fails

echo "--- Installing Frontend Dependencies ---"
cd frontend
npm install

echo "--- Building Angular Frontend ---"
ng build --configuration production

echo "--- Starting Go Backend ---"
echo "--- Opening Browser at http://localhost:5000 in 3 seconds... ---"

# Open browser in a background process
(sleep 2 && { open "http://localhost:5000" 2>/dev/null || xdg-open "http://localhost:5000" 2>/dev/null || explorer.exe "http://localhost:5000"; }) &

cd ../backend
go run main.go