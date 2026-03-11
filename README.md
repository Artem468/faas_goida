# FAAS goida

## Требования
- Go 1.22+
- Docker (для Postgres + rustfs)

## Быстрый старт
1. Создайте `.env` из `.env.example` и при необходимости поправьте значения.
2. Запустите `docker-compose up --build`.
3. API доступен на `http://localhost:8080`.

## API
Все эндпоинты, кроме `/auth/*`, требуют `Authorization: Bearer <token>`.

### Аутентификация
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`

### Проекты
- `POST /projects`
- `GET /projects`
- `GET /projects/{id}`
- `PUT /projects/{id}`
- `DELETE /projects/{id}`

### Файлы
- `POST /projects/{project_id}/files` (multipart: `name` опционально, `file` обязательно)
- `GET /projects/{project_id}/files`
- `GET /projects/{project_id}/files/{id}`
- `PUT /projects/{project_id}/files/{id}` (multipart: `name` обязательно, `file` опционально)
- `DELETE /projects/{project_id}/files/{id}`

### Выполнение
- `POST /call`
  - Тело: `{ "project_id": 1, "file_id": 10 }`
  - Запускает `./bin/goida_lang run <скачанный_файл>`
  - Содержимое файла кешируется в памяти по S3‑ключу на 24 часа

## Примечания
- Для имени временного файла используется сохраненное имя (с `.goida` по умолчанию).
- При обновлении файла и смене S3‑ключа он будет заново скачан и закеширован отдельно.