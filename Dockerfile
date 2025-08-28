FROM python:3.10-slim

RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir --upgrade pip

WORKDIR /app

COPY . .

RUN pip install --no-cache-dir -r requirements.txt
RUN pip install --no-cache-dir piper-tts onnxruntime-gpu

ENV PIPER_DEVICE=cuda

EXPOSE 5000

CMD ["python", "app.py"]
