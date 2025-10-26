## Локальная разработка

**Устанавливаем необходимые библиотеки**

**Клиент для Kafka**

`go get github.com/segmentio/kafka-go`

**Драйвер для PostgreSQL**

`go get github.com/jackc/pgx/v5`

**Для загрузки переменных окружения (опционально)**

`go get github.com/joho/godotenv`

**Запуск контейнеров Postgres и Kafka**

Теперь локально работает кластер Kafka из трех брокеров. Можно подключаться к нему, используя bootstrap-серверы: localhost:9092,localhost:9094,localhost:9096

**Запустить Go-приложение**

`go run cmd/main.go`

**Отправьте тестовое сообщение в Kafka**

Запустите скрипт publisher.sh

1. Для работы скрипта понадобится утилита jq. Это стандартный инструмент для работы с JSON в командной строке.
   Windows (с помощью Chocolatey): 

    ```choco install jq```
       
    macOS (с помощью Homebrew):
    
    ```brew install jq```
       
    Linux (Debian/Ubuntu): 
    
    ```sudo apt-get install jq```

2. Сделайте скрипт исполняемым (в терминале Git Bash, WSL или на Linux/macOS):

   chmod +x publisher.sh

3. Запустите скрипт из корневой папки:

   ```./publisher.sh```

**Фронтенд сервиса доступен по адресу http://localhost:8080/**