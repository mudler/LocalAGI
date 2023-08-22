#!/bin/bash

pip uninstall hnswlib chromadb-hnswlib -y
pip install hnswlib chromadb-hnswlib
cd /app
python3 /app/main.py