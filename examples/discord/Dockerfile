FROM python:3.10-bullseye
WORKDIR /app
COPY ./requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY . /app

ENTRYPOINT [ "python", "./main.py" ];
