# exo

Инструмент для проксирования локальных сервисов на публичный сервер.

## Быстрый старт

### 1. Скачай бинарник

```bash
# macOS (Apple Silicon)
curl -L https://github.com/nvsces/exo/releases/latest/download/exo-darwin-arm64 -o /usr/local/bin/exo && chmod +x /usr/local/bin/exo

# macOS (Intel)
curl -L https://github.com/nvsces/exo/releases/latest/download/exo-darwin-amd64 -o /usr/local/bin/exo && chmod +x /usr/local/bin/exo

# Linux
curl -L https://github.com/nvsces/exo/releases/latest/download/exo-linux-amd64 -o /usr/local/bin/exo && chmod +x /usr/local/bin/exo
```

Или собери из исходников:

```bash
git clone https://github.com/nvsces/exo.git
cd exo
go build -o exo .
```

### 2. Подключи свой локальный сервис

Допустим, у тебя локально запущено приложение на порту `3000`:

```bash
exo connect --server example.com --subdomain ivan --local 3000
```

Готово! Твоё приложение доступно по адресу:

```
http://ivan.example.com:8080
```

### 3. Поделись ссылкой

Отправь ссылку `http://ivan.example.com:8080` коллеге, заказчику или тестировщику — они увидят твой локальный сервис.

## Команда `connect`

```bash
exo connect --server example.com --subdomain <имя> --local <порт>
```

| Флаг | Описание | По умолчанию |
|------|----------|-------------|
| `--server` | Адрес сервера | `localhost` |
| `--subdomain` | Имя поддомена (обязательно) | — |
| `--local` | Порт локального сервиса | `3000` |
| `--reconnect` | Автоматическое переподключение | `true` |

### Примеры

```bash
# React/Next.js (порт 3000)
exo connect --server example.com --subdomain frontend --local 3000

# Backend API (порт 8000)
exo connect --server example.com --subdomain api --local 8000

# Несколько сервисов одновременно — просто запусти в разных терминалах
exo connect --server example.com --subdomain web --local 3000
exo connect --server example.com --subdomain api --local 8000
```

## Как это работает

```
Браузер → http://ivan.example.com:8080
               ↓
         Сервер (VPS)
               ↓ (TCP туннель)
         Твой компьютер
               ↓
         localhost:3000
```

Клиент устанавливает постоянное TCP-соединение с сервером. Когда кто-то открывает `ivan.example.com:8080` в браузере, сервер пробрасывает запрос через туннель на твой `localhost:3000` и возвращает ответ.

## Установка сервера

### 1. Скачай бинарник на VPS

```bash
curl -L https://github.com/nvsces/exo/releases/latest/download/exo-linux-amd64 -o /usr/local/bin/exo && chmod +x /usr/local/bin/exo
```

### 2. Создай systemd-сервис

```bash
sudo nano /etc/systemd/system/exo.service
```

```ini
[Unit]
Description=exo tunnel server
After=network.target

[Service]
ExecStart=/usr/local/bin/exo server --domain example.com --port 8080 --tunnel-port 9000
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

### 3. Запусти

```bash
sudo systemctl daemon-reload
sudo systemctl enable exo
sudo systemctl start exo
```

### Управление

```bash
sudo systemctl status exo    # статус
sudo journalctl -u exo -f    # логи в реальном времени
sudo systemctl restart exo   # перезапуск
```

## FAQ

**Поддомен занят?**
Выбери другой: `--subdomain ivan2`. Каждый разработчик должен использовать уникальное имя.

**Соединение оборвалось?**
Клиент переподключится автоматически через 3 секунды.

**Можно несколько сервисов?**
Да, запусти несколько `exo connect` с разными `--subdomain` и `--local`.

**Не работает?**
Проверь что локальный сервис запущен: `curl http://localhost:3000`
