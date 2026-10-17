# Етап 10 — Full-Stack Go: HTML-шаблони, HTMX та go:embed (звіт після рефакторингу)

## Мета
Перетворити захищений REST API LibraTrack на веб-застосунок із серверним рендерингом, живим пошуком і видаленням рядків через HTMX (без JS-фреймворків) та зібрати все в один портативний бінарник через `go:embed`. Вимоги — `HomeWork 10.md`.

## Що змінено в рефакторингу
Первинну реалізацію звірено з текстом ДЗ пункт за пунктом. Вимоги відповідали, але виявлено й закрито три прогалини:
1. **Не було автотесту пошуку на PostgreSQL** (у попередньому звіті це було обмеженням). Додано `TestPostgresBookSearchAndBatchAvailability` ([integration_test.go](../internal/repository/integration_test.go)): пошук за назвою (без урахування регістру), ISBN, автором; `%` трактується буквально; SQL-метасимволи (`'; DROP TABLE books;--`) нешкідливі; `AvailabilityByBooks` повертає коректні total/available і нулі для книги без копій.
2. **Невдале видалення було «мовчазним»**: htmx не обробляє 4xx, тож 409 (книга з історією позик), 403, 404 нічого не показували. Додано [app.js](../web/static/js/app.js) (зовнішній файл — CSP `default-src 'self'` дотримано) з обробником `htmx:responseError` і блок `<div id="flash" role="alert" hidden>` у [layout.html](../web/templates/layout.html) (+ стилі). Тест макета перевіряє наявність `app.js` і `#flash`.
3. **Неточність у звіті**: «3 файли html/template» — це 3 *Go*-файли з імпортом `html/template`; самих `.html`-шаблонів 4 (layout, catalog, login, book_row). Виправлено нижче.

## Частина 1 — ручні завдання
### Завдання 1.1 — архітектура веб-шару та аудит автоекранування
- **Пп. 1 (cookie/XSS)** — [design_v10.txt](../design_v10.txt), розділ 1: перехід з `Authorization: Bearer` (урок 9) на cookie `libratrack_token` (`HttpOnly`, `SameSite=Lax`, `Secure` за TLS або `COOKIE_SECURE=true`, `Max-Age=86400`); токен ніколи не потрапляє в тіло відповіді. Пояснено, чому `localStorage` небезпечний: доступний будь-якому JS на origin → один XSS дає повне й відтерміноване захоплення сесії (24 год) з будь-якої машини; HttpOnly-cookie скриптам недоступний; серверно-рендереному застосунку JS для зберігання токена не потрібен, тож `localStorage` — чиста поверхня атаки. Окремо описано ціну переходу (CSRF) і заходи: `SameSite=Lax`, обов'язковий `HX-Request` для DELETE, CSP; залишковий ризик login-CSRF.
- **Пп. 2 (html/template)** — пошук `(html|text)/template` по всіх `*.go`: 3 збіги (`web/templates.go`, `internal/app/library.go`, `internal/handler/web/catalog.go`), усі `html/template`; `text/template` відсутній.
- **Пп. 3 (template.HTML)** — `template.HTML/JS/URL/CSS/HTMLAttr/JSStr/Srcset` у `web/`, `internal/`, `cmd/`: 0 використань. Усі недовірені значення (title, authors, category, isbn, `q`, повідомлення) — лише через `{{.Field}}`; inline `<script>`, `<style>`, `style=`, `on*=` відсутні. Регресійний тест: назва `<script>alert(1)</script>` рендериться як `&lt;script&gt;…`. Порушень немає; повний протокол — розділ 3 `design_v10.txt`.

### Завдання 1.2 — ізоляційний тест єдиного бінарника
Виконано повторно в цьому сеансі:
1. `CGO_ENABLED=0 go build -o libratrack.exe ./cmd/api` → 27 315 712 байт.
2. Чиста тимчасова папка `%TEMP%\verify_libratrack.*` містила **лише** `libratrack.exe` (без `web/`, `templates/`, `migrations/`).
3. Запуск звідти з PostgreSQL (окрема схема `libratrack_verify`): міграції застосувались із вбудованих файлів, сервер стартував.
4. HTTP-перевірка (curl): `/catalog` → 200 `text/html`; `/static/css/style.css` → 200 `text/css`; `/static/js/htmx.min.js` → 200 `text/javascript` (48 101 байт); `/static/js/app.js` → 200; `/web/books?q=` з `HX-Request` → лише `<tr>`.
5. Наскрізний сценарій: register → 201; login → 303 + HttpOnly-cookie; reader бачить 0 кнопок Delete і отримує **403** на DELETE; анонім → **403**; після підвищення до librarian — 10 кнопок Delete, DELETE без `HX-Request` → **403**, з `HX-Request` → **200 з порожнім тілом**, повторно → **404**.
6. Прибирання: сервер зупинено, схему видалено, бінарник і тимчасову папку прибрано.

**Відхилення від буквального тексту ДЗ:** перевірено через curl, а не «в браузері» (див. «Не перевірено»).

## Частина 2 — промпти (`prompts.txt`)
### Завдання 2.1 — layout, catalog, book_row
Промпт «Task 2.1 (Lesson 10)» у [prompts.txt](../prompts.txt). Результат:
- [layout.html](../web/templates/layout.html): HTML5, `{{block "content" .}}{{end}}`, `/static/css/style.css` і локальний htmx, блок входу/виходу, flash-блок.
- [catalog.html](../web/templates/catalog.html): таблиця книг, форма пошуку (`role="search"`, `<label for>`, працює без JS через GET `/catalog?q=`).
- [partials/book_row.html](../web/templates/partials/book_row.html): `book_row` та `book_rows` (з «No books found.»); індикатор доступності — бейдж із текстом «Available (x/y)» / «Unavailable» (не лише колір).

### Завдання 2.2 — HTMX (живий пошук і видалення)
Промпт «Task 2.2 (Lesson 10)» у `prompts.txt`. Атрибути точно за ДЗ:
- Пошук: `hx-get="/web/books"`, `hx-trigger="keyup changed delay:300ms"`, `hx-target="#book-list"`.
- Видалення (кнопка лише для librarian/admin): `hx-delete="/web/books/{{.ID}}"`, `hx-target="closest tr"`, `hx-swap="outerHTML"`, `hx-confirm="Delete this book?"`.
- Обробники в [internal/handler/web/catalog.go](../internal/handler/web/catalog.go): `GET /web/books` повертає лише `book_rows` при `HX-Request: true`, інакше 303 на `/catalog`; `DELETE /web/books/:id` — лише librarian/admin і лише з `HX-Request`, успіх = **порожній `200 OK`**; невідомий id → 404, некоректний → 400, FK-конфлікт → 409 (тепер із повідомленням у UI).
- Пошук у репозиторії: `ILIKE` за назвою, ISBN, автором; `%`/`_` екрануються; запит параметризований; доступність усіх книг сторінки — одним згрупованим запитом (без N+1).
- Журнал промптів: `prompts.txt` містить промпти 2.1, 2.2, 3.1, 3.2 та запис цього рефакторингу; решта — промпти попереднього уроку.

## Частина 3 — агентна робота
### Завдання 3.1 — go:embed
- [web/templates.go](../web/templates.go): `//go:embed templates`, `embed.FS`, `template.ParseFS` (не `ParseGlob`), окремий набір шаблонів на сторінку (блоки `content` не конфліктують).
- [web/static.go](../web/static.go): `//go:embed static` + `fs.Sub(staticFS, "static")` — `/static/css/style.css` відповідає `css/style.css`.
- [migrations/embed.go](../migrations/embed.go) + `ApplyMigrationsFS` (golang-migrate `iofs`): понад вимоги ДЗ, інакше бінарник у чистій папці не знайшов би `./migrations`; `MIGRATIONS_PATH` — необов'язкове перевизначення.
- Підключення в [cmd/api/main.go](../cmd/api/main.go): `app.NewLibraryRouter(db, app.WithWeb(pages, staticFS))`.

### Завдання 3.2 — табличні тести з httptest
[internal/handler/web/catalog_test.go](../internal/handler/web/catalog_test.go) (реальні вбудовані шаблони, фейкові сховища):
1. `GET /catalog` → 200, розмітка макета (`<!doctype html>`, `LibraTrack`, `app.js`, `#flash`), `id="book-list"`, таблиця.
2. `GET /web/books?q=Go` з `HX-Request: true` → лише `<tr>`-фрагменти, без `<html`/`<table`.
3. `DELETE` — анонім, невалідний токен, reader, librarian без `HX-Request` → **403**, нічого не видалено; librarian/admin → порожній 200; 404; 400.

Додатково: автоекранування, CSP/nosniff, логін ставить HttpOnly+SameSite=Lax cookie без JWT у тілі, редірект без `HX-Request`.

## Самоперевірка за критеріями ДЗ
| Критерій | Статус |
|---|---|
| 1. Інтерактивність UI | Розмітка й сервер перевірені (HTTP + тести); поведінку в браузері — не перевірено |
| 2. Єдиний бінарник | Пройдено (чиста папка, лише exe) |
| 3. Безпека (HttpOnly JWT, автоекранування) | Пройдено, аудит задокументовано |
| 4. Тести (фрагменти, RBAC) | Пройдено |
| 5. Журнал промптів | `prompts.txt` |

## Перевірки (цей сеанс)
- `gofmt -l internal web cmd migrations` — порожньо; `go vet ./...` — чисто; `go build ./...` — ок.
- `go test -race -count=1 ./...` — усі пакети `ok` (`TEST_DATABASE_URL` налаштовано, PostgreSQL-тести виконувались, зокрема новий тест пошуку).

## Не перевірено / обмеження
- **Браузер**: debounce 300 мс, діалог `hx-confirm`, зникнення рядка без перезавантаження і новий flash-обробник `app.js` не запускались у реальному браузері (перевірено лише HTTP-рівень, розмітку, атрибути та відповіді сервера). Рекомендовано пройти вручну: запустити бінарник і відкрити `/catalog`.
- CSRF-токенів немає: захист — `SameSite=Lax` + `HX-Request` для DELETE + CSP; login-CSRF не закрито.
- `/catalog` без пагінації (до 100 книг); каталог доступний анонімно (як і `GET /api/v1/books`).
- `cmd/libratrack/main.go` не змінювався і веб-шар не підключає (веб — через `cmd/api`).
- Файли `db.go`, `main.go`, `.github/workflows/theme8-check.yml` у корені були неперевіреними до цього етапу й не чіпались.

## Результат
Усі підпункти ДЗ 10 відповідають вимогам викладача; прогалини (тест пошуку на PostgreSQL, мовчазні помилки видалення, неточність звіту) закрито. Залишається ручна браузерна перевірка.
