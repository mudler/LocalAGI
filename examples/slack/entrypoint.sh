#!/bin/bash

cd /app

pip uninstall hnswlib -y

git clone https://github.com/nmslib/hnswlib.git
cd hnswlib
pip install .
cd ..

python main.py