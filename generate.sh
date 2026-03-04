#!/bin/bash
set -e

echo "🚀 Генерация кода из OpenAPI..."

OPENAPI_FILE="api/openapi.yaml"

if [ ! -f "$OPENAPI_FILE" ]; then
    echo "❌ Файл $OPENAPI_FILE не найден!"
    echo "Создай его: mkdir -p api && cp путь/к/openapi.yaml api/"
    exit 1
fi

# Создаём папку для сгенерированного кода
mkdir -p pkg/gen

echo "📦 Запускаем openapi-generator в Docker..."

docker run --rm \
  -v "$(pwd):/local" \
  openapitools/openapi-generator-cli generate \
  -i /local/$OPENAPI_FILE \
  -g go-server \
  -o /local/pkg/gen \
  --package-name gen \
  --additional-properties=enumClassPrefix=false,hideGenerationTimestamp=true

echo ""
echo "✅ Код успешно сгенерирован!"
echo "📁 Папка: pkg/gen/"
echo ""
echo "💡 Совет: добавь в .gitignore:"
echo "   pkg/gen/"