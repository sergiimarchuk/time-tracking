# Сохраним токен в переменную
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo0LCJ1c2VybmFtZSI6InRlc3RhcGkiLCJleHAiOjE3NjQwMDg2NzcsImlhdCI6MTc2MzkyMjI3N30.e2gxIWxR1ub_MEujAcC55nKwimx9tNk6g4cDEzLSGdI"

# 1. Создать запись
curl -X POST http://localhost:8080/api/v1/worklogs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "date": "2025-11-23",
    "description": "Тестирование REST API",
    "hours": 2.5
  }'

# 2. Получить все записи
curl -X GET http://localhost:8080/api/v1/worklogs \
  -H "Authorization: Bearer $TOKEN"

# 3. Получить статистику
curl -X GET http://localhost:8080/api/v1/stats \
  -H "Authorization: Bearer $TOKEN"

# 4. Обновить запись (замени ID на реальный)
curl -X PUT http://localhost:8080/api/v1/worklogs/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "date": "2025-11-23",
    "description": "Обновлённое описание",
    "hours": 3.0
  }'

# 5. Удалить запись
curl -X DELETE http://localhost:8080/api/v1/worklogs/1 \
  -H "Authorization: Bearer $TOKEN"
