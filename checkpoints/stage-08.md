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
- Додано PostgreSQL інтеграційне порівняння наївного N+1 завантаження проти production `Preload`: 10 книг, 5 авторів, 3 категорії; логування SQL, перевірка пакетних запитів, 7 замірів медіани. Також доданий ізольований тест `up/down/up` для legacy-міграції, який перевіряє порожню категорію та точне відновлення авторського/категорійного тексту.
- Виправлено legacy migration: порожня категорія мапиться на `Uncategorized`; оригінальні `author` / `category` текстові значення зберігаються для точного відновлення при `down`. Legacy `author` залишається comma-separated форматом, тому кома в імені неоднозначна як роздільник.
- Після виправлення очікувань PostgreSQL SELECT count повний запуск `go test ./... -count=1` пройшов. `TestPostgresHomeworkNormalizationMigrationRoundTrip`, `TestPostgresBookListUsesConstantPreloadQueries` і `TestPostgresBookListNPlusOneComparison` пройшли на тестовій БД.
- Фактичне порівняння на 10 книгах: naive — 21 SELECT, медіана 2.796 ms; GORM Preload — 4 SELECT, медіана 1.042 ms. Preload складається з запитів до `books`, `book_authors`, `authors` і `categories`; пакетні зв'язки `book_authors` та автори завантажуються через `IN`. API smoke test окремо не запускався.
- Раніше також пройшли `go test -race ./...`, `go vet ./...`, `go build ./...`, `go mod tidy -go=1.21` і `git diff --check`; актуальний стан цих перевірок після останніх тестових змін не переперевірявся.
- Структура зберігає `module homework` та старі імпорти; Theme 8 workflow з вимогою `module libratrack` залишається несумісним без окремого рішення щодо старого контракту.

## Статус
Рефакторинг, перевірка legacy-міграції та N+1 порівняння підтверджені PostgreSQL інтеграційними тестами; повний `go test ./... -count=1` пройшов, фактичні SELECT counts і latency зафіксовані в README. API smoke test залишається незапущеним.
