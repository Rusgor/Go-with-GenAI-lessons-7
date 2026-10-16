# Етап 10 — Full-Stack Go: HTML-шаблони, HTMX та go:embed

## Мета
Перетворити захищений REST API LibraTrack на веб-застосунок із серверним рендерингом, живим пошуком і видаленням рядків через HTMX (без JS-фреймворків) та зібрати все в один портативний бінарник через `go:embed`.

## Реалізовано
- **Шаблони** (`html/template`): `web/templates/layout.html` (`{{block "content" .}}`), `catalog.html`, `login.html`, `partials/book_row.html` (`book_row` + `book_rows`), стилі `web/static/css/style.css`, `htmx.min.js` 1.9.12 локально в `web/static/js/`.
- **Каталог**: таблиця книг, індикатор доступності (бейдж із текстом «Available (x/y)» / «Unavailable»), форма пошуку з fallback без JS.
- **HTMX**: пошук (`hx-get="/web/books"`, `hx-trigger="keyup changed delay:300ms"`, `hx-target="#book-list"`); видалення (`hx-delete`, `hx-target="closest tr"`, `hx-swap="outerHTML"`, `hx-confirm="Delete this book?"`) — кнопка рендериться лише для librarian/admin. Обробник видалення відповідає порожнім `200 OK`.
- **Обробники** `internal/handler/web/catalog.go`: `GET /`, `/catalog`, `/login`, `POST /login`, `/logout`, `GET /web/books`, `DELETE /web/books/:id`. Фрагмент повертається лише при `HX-Request: true`, інакше — 303 на `/catalog`.
- **JWT у HttpOnly cookie** `libratrack_token` (`HttpOnly`, `SameSite=Lax`, `Secure` за TLS або `COOKIE_SECURE=true`); новий `OptionalCookieAuth`, спільна `parseClaims` з Bearer-middleware. JSON API не змінено.
- **RBAC**: `DELETE /web/books/:id` — лише librarian/admin і лише з `HX-Request: true`; анонім/reader → **403**.
- **Додатковий захист**: CSP `default-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'`, `nosniff`, `Referrer-Policy`; htmx з `allowEval:false`.
- **Репозиторій**: пошук `q` (ILIKE за назвою, ISBN, автором; `%`/`_` екрануються, запит параметризований), `AvailabilityByBooks` одним згрупованим запитом (без N+1).
- **go:embed**: `web/templates.go` (`embed.FS` + `template.ParseFS`), `web/static.go` (`fs.Sub(staticFS, "static")`), `migrations/embed.go` + `ApplyMigrationsFS` (golang-migrate `iofs`); підключено в `cmd/api/main.go` через `app.NewLibraryRouter(db, app.WithWeb(pages, staticFS))`. `MIGRATIONS_PATH` — необов'язкове перевизначення.
- **Документи**: [design_v10.txt](../design_v10.txt) (архітектура cookie, чому localStorage небезпечний, аудит автоекранування), [prompts.txt](../prompts.txt) (промпти Завдань 2.1, 2.2, 3.1, 3.2).

## Аудит автоекранування (Завдання 1.1)
- `text/template` відсутній; у проєкті лише `html/template` (3 файли).
- `template.HTML/JS/URL/CSS/HTMLAttr` — 0 використань; inline `<script>`, `<style>`, `style=`, `on*=` у шаблонах немає.
- Усі недовірені значення виводяться через `{{.Field}}`; регресійний тест перевіряє, що `<script>alert(1)</script>` у назві книги рендериться екранованим. Порушень не знайдено.

## Ізоляційний тест бінарника (Завдання 1.2)
- `CGO_ENABLED=0 go build -o libratrack.exe ./cmd/api` → 27 314 688 байт; скопійовано **лише** exe в `%TEMP%\verify_libratrack` (без `web/`, `templates/`, `migrations/`).
- Запуск звідти з PostgreSQL (окрема схема `libratrack_verify` у тестовій БД): міграції застосувались із вбудованих файлів, сервер стартував.
- Перевірено HTTP-запитами (не в браузері): `/catalog` → 200 `text/html`; `/static/css/style.css` → 200 `text/css`; `/static/js/htmx.min.js` → 200 `text/javascript` (48 101 байт); `/web/books?q=Go` з `HX-Request` → 200 (фрагмент).
- Наскрізно (curl): register → 201; login → 303 + cookie з прапорцем `#HttpOnly_`; reader `DELETE` → **403**; після підвищення до librarian `DELETE /web/books/1` → **200 з порожнім тілом**; повторне → 404; пошук `q=practical` повернув «Practical PostgreSQL». Після перевірки схему видалено, тимчасові файли та бінарник прибрано.

## Перевірки
- `gofmt -l internal web cmd migrations` — порожньо; `go vet ./...` — пройшов; `go build ./...` — пройшов.
- `go test -race -count=1 ./...` — пройшов (`TEST_DATABASE_URL` налаштовано, PostgreSQL-інтеграційні тести виконувались).
- Нові табличні тести `internal/handler/web/catalog_test.go`: `GET /catalog` (розмітка макета + таблиця), HTMX-фрагмент `q=Go` (лише `<tr>`), порожній результат, кнопки видалення за ролями, автоекранування, security-заголовки, redirect без `HX-Request`; `DELETE` — анонім/невалідний токен/reader/librarian без `HX-Request` → 403 (нічого не видалено), librarian/admin → порожній 200, 404, 400; логін ставить HttpOnly+SameSite=Lax cookie і не віддає JWT у тілі.

## Не перевірено / обмеження
- Візуальна перевірка в **браузері** та реальна взаємодія HTMX (debounce 300 мс, діалог `hx-confirm`, зникнення рядка без перезавантаження) не виконувались — перевірено лише HTTP-рівень і розмітку з атрибутами. Рекомендовано пройти вручну: запустити бінарник і відкрити `/catalog`.
- Для пошуку (`ILIKE`) окремого автоматичного PostgreSQL-тесту не додано; працездатність підтверджено ручним запитом `q=practical` до живої БД.
- CSRF-токенів немає: захист — `SameSite=Lax` + обов'язковий `HX-Request` для DELETE + CSP. Login-CSRF не закривається (див. design_v10.txt).
- Видалення книги з історією позик поверне 409 (обмеження FK), UI-повідомлення для цього не додано.
- Сторінка `/catalog` без пагінації: показується до 100 книг.
- Каталог доступний анонімно (як і `GET /api/v1/books`); `cmd/libratrack/main.go` не змінювався і веб-шар не підключає.
- Файли `db.go`, `main.go`, `.github/workflows/theme8-check.yml` у корені були неперевіреними ще до цього етапу й не чіпались.

## Результат
Усі три частини ДЗ реалізовано; автоматичні перевірки та ізоляційний запуск бінарника пройшли. Ручна браузерна перевірка інтерактивності залишається за вами.
