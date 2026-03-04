FROM python:3.12-slim

WORKDIR /app

# Копируем зависимости отдельно для кэширования Docker слоёв
COPY requirements.txt .
RUN pip install --no-cache-dir --trusted-host pypi.org --trusted-host files.pythonhosted.org -r requirements.txt

COPY . .

EXPOSE 8000

# --host 0.0.0.0 — слушаем на всех интерфейсах (нужно для Docker)
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
