#!/bin/bash
set -e # Exit immediately if a command fails

echo "--- Installing Frontend Dependencies ---"
cd frontend
npm install

echo "--- Building Angular Frontend ---"
ng build --configuration production

echo "--- Starting Go Backend ---"
cd ../backend
go run main.go