# На сервере
docker exec -it worklog-tracker sh

# Внутри контейнера смотрим где база
ls -la /app/
ls -la /app/data/
ls -la /root/

# Проверяем переменные окружения
env | grep DATABASE

# Выходим
exit
