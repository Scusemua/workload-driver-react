#!/bin/bash

echo "PUBLIC_PATH=/" >> .development.env
echo "ASSET_PATH=/" >> .development.env
echo "JUPYTER_ADDRESS=localhost" >> .development.env
echo "JUPYTER_PORT=8888" >> .development.env

echo "PUBLIC_PATH=/dashboard" >> .production.env
echo "ASSET_PATH=/dashboard" >> .production.env
echo "JUPYTER_ADDRESS=host.docker.internal" >> .production.env
echo "JUPYTER_PORT=8888" >> .production.env
