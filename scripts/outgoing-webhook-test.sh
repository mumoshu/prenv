#!/bin/env bash

curl -X POST http://localhost:8080/ \
   -H "Content-Type: application/x-www-form-urlencoded" \
   -d "param1=value1&param2=value2"
