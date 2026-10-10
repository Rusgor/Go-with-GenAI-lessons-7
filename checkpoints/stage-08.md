# Етап 8 — LibraTrack API та локальна перевірка

## Мета
Підключити HTTP API до нових GORM репозиторіїв окремо від наявного `devices` API.

## Виконано
- Додано Gin router із `/health`, Swagger endpoint, CORS і REST маршрутами категорій, авторів, книг, копій та доступності.
- Реалізовано валідацію, пагінацію/фільтр `category_id`, повну заміну книги та безпечний JSON error envelope.
- Додано команду `cmd/libratrack`; також `cmd/api` зберігає поточний запуск in-memory `devices` за замовчуванням і запускає LibraTrack лише за наявності `DATABASE_URL`.
- Додано handler/app/model тести для валідації, безпечних помилок, порожнього списку, health, Swagger та CORS.

## Рефакторинг та перевірки
- Залишено окремий LibraTrack API, щоб не змінювати Lesson 7 `devices` контракт та його in-memory поведінку.
- `GormBookRepository.Delete` має виконувати запит із переданим `context.Context`; додано інтеграційне покриття скасованого контексту.
- Додано PostgreSQL інтеграційне порівняння наївного N+1 завантаження проти production `Preload`: 10 книг, 5 авторів, 3 категорії; логування SQL, перевірка пакетних запитів, 7 замірів медіани. Це відтворюваний тестовий стенд, а не вже отримані докази.
- Виправлено legacy migration: порожня категорія мапиться на `Uncategorized`; оригінальні `author` / `category` текстові значення зберігаються для точного відновлення при `down`. Legacy `author` залишається comma-separated форматом, тому кома в імені неоднозначна як роздільник.
- Після змін пройшли `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod tidy -go=1.21` і `git diff --check`.
- `TestPostgresBookListNPlusOneComparison` явно пропущений, бо `TEST_DATABASE_URL` не заданий. API smoke test, SQL logs і latency на реальній БД не зафіксовані: локальний PostgreSQL недоступний, Docker/Podman/psql/migrate відсутні.
- Структура зберігає `module homework` та старі імпорти; Theme 8 workflow з вимогою `module libratrack` залишається несумісним без окремого рішення щодо старого контракту.

## Статус
Рефакторинг та відтворюваний тест порівняння додані, але перевірка міграцій на PostgreSQL і фактичні SQL/latency докази для README залишаються незавершеними. Не вважати всі вимоги здачі підтвердженими, доки тест не запущено на реальній БД.
