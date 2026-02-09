#!/bin/sh
# Wrapper script for running Hamster Wheel in dev mode with Vite dev server
export FRONTEND_DEVSERVER_URL=http://localhost:5173
exec go run .
