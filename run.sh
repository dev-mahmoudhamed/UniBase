#!/bin/bash
set -e # Exit immediately if a command fails

echo "--- Building Angular Frontend ---"
cd frontend && ng build --configuration production

echo "--- Starting Go Backend ---"
cd ../backend && go run main.go