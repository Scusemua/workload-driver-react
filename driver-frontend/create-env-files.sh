#!/bin/bash

echo "PUBLIC_PATH=/" >> .development.env
echo "ASSET_PATH=/" >> .development.env

echo "PUBLIC_PATH=/notebook-dashboard" >> .production.env
echo "ASSET_PATH=/notebook-dashboard" >> .production.env
