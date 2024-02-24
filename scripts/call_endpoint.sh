#!/bin/bash

while true; do
  curl http://localhost:8080/eth/balance/0x00a3Ac5E156B4B291ceB59D019121beB6508d93D
  # Sleep for 500 milliseconds
  sleep 0.1
done