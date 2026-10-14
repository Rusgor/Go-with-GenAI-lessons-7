# Домашнє завдання: REST API на Go

Ви реалізуєте REST API для **власної сутності** (ваш варіант). Усі варіанти мають **однаковий контракт**,
тому правильність перевіряється автоматичними тестами. Ви пишете сервіс самі (з ШІ-помічником чи без),
але маєте розуміти кожен рядок свого коду: це перевіряється на захисті.

## 1. Як почати

1. Створіть свій репозиторій із цього шаблону (кнопка **Use this template** або посилання від викладача).
2. Відкрийте файл `VARIANT` і запишіть назву **вашого варіанта** (див. таблицю нижче), наприклад `movies`.
3. Запустіть `go test ./...`. Усі тести спочатку **червоні**, це нормально: ви рухаєтеся до зеленого.
4. Реалізуйте `NewRouter()` у `internal/app/app.go` й усе, що потрібно для нього.
5. Після кожного `git push` GitHub Actions запускає перевірку й показує бали на сторінці запуску (розділ *Summary*).

> Тести звертаються до вашого API **лише** через `app.NewRouter() http.Handler`. Структуру решти коду, роутер
> (`net/http`, `gin`, `chi`...) і бібліотеки обираєте ви. Файли `*_test.go`, `scripts/` і `.github/workflows/ci.yml`
> **не змінюйте**: під час оцінювання використовуються оригінальні версії.

## 2. Варіанти

Кожна сутність має **4 обов'язкові рядкові поля**, **одне необов'язкове ціле** і **одне необов'язкове текстове** поле.
Четверте обов'язкове поле є полем **фільтра**.

| Варіант (`VARIANT`) | Ресурс у URL | Обов'язкові поля (4-те це фільтр) | Необов'язкове ціле | Необов'язковий текст |
|---|---|---|---|---|
| `books` | `books` | `title`, `isbn`, `author`, `category` | `published_year` | `description` |
| `movies` | `movies` | `title`, `director`, `country`, `genre` | `release_year` | `synopsis` |
| `tasks` | `tasks` | `title`, `assignee`, `project`, `status` | `priority` | `notes` |
| `recipes` | `recipes` | `name`, `author`, `difficulty`, `cuisine` | `cook_minutes` | `instructions` |
| `devices` | `devices` | `name`, `serial`, `manufacturer`, `type` | `warranty_months` | `comment` |
| `courses` | `courses` | `title`, `teacher`, `level`, `subject` | `hours` | `program` |
| `albums` | `albums` | `title`, `artist`, `label`, `genre` | `release_year` | `notes` |
| `pets` | `pets` | `name`, `owner`, `breed`, `species` | `age` | `notes` |

Нижче в прикладах використано варіант `books`. Підставте своє.

## 3. Контракт API

Усі відповіді (крім `204`) мають `Content-Type: application/json`. Ключі JSON пишуться в `snake_case`.

| Метод і шлях | Успіх | Помилки |
|---|---|---|
| `GET /health` | `200` `{"status":"ok"}` | |
| `POST /api/v1/books` | `201` + створений ресурс | `422` |
| `GET /api/v1/books` | `200` + масив | `400` (`invalid_pagination`) |
| `GET /api/v1/books/{id}` | `200` + ресурс | `400` (`invalid_id`), `404` |
| `PUT /api/v1/books/{id}` | `200` + збережений ресурс | `400`, `422`, `404` |
| `DELETE /api/v1/books/{id}` | `204` без тіла | `400`, `404` |

Невідомий маршрут дає `404`, непідтримуваний метод (наприклад `PATCH`) дає будь-який `4xx`.

### Ресурс

```json
{
  "id": 1,
  "title": "The Go Programming Language",
  "isbn": "9780134190440",
  "author": "Donovan, Kernighan",
  "category": "programming",
  "published_year": 2015,
  "description": null,
  "created_at": "2026-10-05T12:00:00.123456Z",
  "updated_at": "2026-10-05T12:00:00.123456Z"
}
```

- `id`: додатне ціле, починається з `1`, зростає на 1, **ніколи не використовується повторно** після видалення.
- Необов'язкові поля **завжди присутні** у відповіді, `null`, якщо не були задані.
- `created_at` і `updated_at`: час у форматі RFC 3339. Створення виставляє обидва. `PUT` змінює лише `updated_at`.

### Валідація (`422`, код `validation_error`)

- Обов'язкове поле відсутнє, `null`, не рядок, порожнє або складається лише з пробілів.
- Необов'язкове поле має хибний тип (рядок замість числа, дробове число замість цілого, число замість тексту).
- Тіло запиту не є коректним JSON-об'єктом (зіпсований JSON, порожнє тіло, масив).

### Ідентифікатор

`{id}` має бути цілим числом від `0` до `4294967295` (`uint32`). Інакше (`abc`, `-1`, `1.5`, завелике число)
це `400` з кодом `invalid_id`. Коректний, але неіснуючий `id` це `404` з кодом `not_found`.
Для `PUT` порядок перевірок такий: **спочатку `id` (400), потім тіло (422), потім існування (404)**.

### `PUT`: повна заміна

Тіло такого ж формату, як при створенні. Обов'язкові поля замінюються, **необов'язкові, яких немає в тілі, стають
`null`**. Відповідь містить збережений ресурс (з тим самим `id` і `created_at`). Запит, відхилений валідацією,
нічого не змінює.

### Список

`GET /api/v1/books?page=1&limit=10&category=programming`

- Порожня колекція повертається як `[]`, а не `null`.
- Порядок: за `id` за зростанням, **завжди однаковий**.
- `page` (за замовчуванням `1`) і `limit` (за замовчуванням `10`) мають бути цілими числами `>= 1`, інакше
  `400` з кодом `invalid_pagination`. Якщо `limit > 100`, використовується `100`. Сторінка за межами даних дає `[]`.
- Фільтр: параметр запиту має **ту саму назву, що й четверте обов'язкове поле** (`category`, `genre`, `status`...).
  Збіг точний і чутливий до регістру. Порожнє значення (`?category=`) означає «без фільтра».
  Фільтр працює разом із пагінацією.
- Дуже великі значення (`page=9223372036854775807`) не повинні призводити до паніки чи `500`.

### Формат помилки

```json
{ "error": { "code": "not_found", "message": "Book not found" } }
```

Коди: `invalid_id`, `invalid_pagination`, `validation_error`, `not_found`, `internal_server_error`.
`message` це непорожній рядок. Відповідь **не повинна розкривати внутрішні деталі** (текст паніки, stack trace,
імена файлів). Паніка в обробнику перетворюється на `500` з кодом `internal_server_error`, а подробиці йдуть у лог.

### CORS, конкурентність, стан

- Кожна відповідь має заголовок `Access-Control-Allow-Origin`. Запит `OPTIONS` дає `204` із
  `Access-Control-Allow-Methods` (зі списком методів, зокрема `POST`).
- Сервіс безпечний для паралельних запитів (перевіряється з `go test -race`).
- Кожен виклик `NewRouter()` створює **нове порожнє сховище** (без глобальних змінних зі станом).

## 4. Етапи (рекомендований порядок)

| Етап | Тести | Що робимо |
|---|---|---|
| 1 | `TestStage1_*` | каркас, `/health`, створення й отримання одного ресурсу |
| 2 | `TestStage2_*` | валідація й помилки (`400`, `404`, `422`) |
| 3 | `TestStage3_*` | `PUT`, `DELETE`, ідентифікатори, час |
| 4 | `TestStage4_*` | список, сортування, пагінація, фільтр |
| 5 | `TestStage5_*` | CORS і конкурентність |

Запуск одного етапу: `go test ./internal/app -run TestStage2 -v`. Повна перевірка: `go test -race ./...`.
Окрім видимих тестів, є **приховані** (граничні випадки з цього опису), які запускає викладач.

## 5. Рекомендована структура

```
cmd/api/main.go              запуск сервера (готовий)
internal/app/app.go          NewRouter(): збирає все докупи
internal/model/              структури ресурсу і запитів
internal/repository/         інтерфейс сховища + реалізація в пам'яті (з мʼютексом!)
internal/handler/            HTTP-обробники
internal/middleware/         логування, відновлення після паніки, CORS
```

Типові помилки, на які варто звернути увагу (їх регулярно допускає згенерований код): випадковий порядок
елементів при обході `map`, повернення вказівників на внутрішні об'єкти сховища (гонка даних), `panic` при від'ємних
межах зрізу в пагінації, відсутній час створення, відповідь `PUT` без `id`, розкриття тексту паніки клієнту.

## 6. Оцінювання (100 балів)

| Частина | Бали |
|---|---|
| Видимі автотести (етапи 1-5) | 40 |
| Приховані автотести | 20 |
| `gofmt` і `go vet` без зауважень | 5 |
| Якість коду: шари, інтерфейс сховища, README, Swagger, історія комітів | 15 |
| Захист (пояснити код, змінити вимогу наживо) | 20 |

Автоматична перевірка у вашому репозиторії показує видимі тести та `gofmt`/`vet` (до 45 балів).

## Реалізація варіанта `devices`

Запуск локального API:

```bash
go run ./cmd/api
```

API реалізує `/health` і `/api/v1/devices`: створення та читання пристроїв, повну заміну через `PUT`, видалення через `DELETE`, а також список із пагінацією та точним фільтром за `type`. Дані зберігаються в пам’яті окремого router instance; сховище захищене `sync.RWMutex`, а CORS middleware обробляє preflight `OPTIONS` і дозволяє origin `*`. Деталі виконання та перевірки наведено у [контрольних точках](checkpoints/README.md).

Перевірки перед здачею:

```bash
go test ./...
go test -race ./...
go vet ./...
```

## Lesson 8 Report

### LibraTrack: запуск

У цьому репозиторії Lesson 8 реалізовано як окремий LibraTrack API; наявний Lesson 7 API пристроїв і його чисте in-memory сховище збережені.

```powershell
Copy-Item .env.example .env
docker compose up -d
$env:DATABASE_URL = "postgres://libratrack:libratrack@localhost:5432/libratrack?sslmode=disable"
$env:MIGRATIONS_PATH = "migrations"
$env:PORT = "8081"
go run ./cmd/libratrack
```

API: `http://localhost:8081/health`, `http://localhost:8081/api/v1/books`; Swagger endpoint: `http://localhost:8081/swagger/index.html`. Міграції застосовуються командою `golang-migrate` під час запуску. Перехідна нормалізаційна міграція, що відповідає імені з умови, розміщена окремо в `migrations-homework/migrations/` і має legacy `000001` перед нею; не запускайте цю послідовність поверх основної схеми.

### Промпт для генерації/перевірки міграції

> Згенеруй для PostgreSQL і golang-migrate файли `000002_normalize_books.up.sql` та `000002_normalize_books.down.sql`, виходячи з legacy таблиці `books(id, title, isbn, author, category, published_year, description, created_at, updated_at)`. Нормалізуй автора й категорію в таблиці `authors(id, full_name, bio)` та `categories(id, name)`: категорія має зв’язок one-to-many з книгами, автори — many-to-many через `book_authors(book_id, author_id)`. Додай явні іменовані зовнішні ключі, `ON DELETE RESTRICT` для категорій та авторів, `ON DELETE CASCADE` для зв’язків книги, індекси на зовнішні ключі, унікальні ключі де це потрібно. Up має перенести наявні текстові значення у нормалізовані таблиці; down має відновити legacy колонки й значення та видалити створені об’єкти у зворотному порядку. Не використовуй `AutoMigrate`. Перевір синтаксис і зворотність для PostgreSQL.

> Уточнення, додані під час рефакторингу: legacy `author` трактується як список авторів із комою-роздільником; точний початковий текст `author` і `category` зберігається для відновлення при down. Порожню/пробільну категорію up відносить до `Uncategorized`. Кома в самому імені автора неоднозначна для такого legacy формату.

### Діагностика N+1 та latency

#### Що реалізовано

`GormBookRepository.List` використовує `.Preload("Authors").Preload("Category")`. Інтеграційний тест `TestPostgresBookListNPlusOneComparison` додає окремий детермінований набір із 10 книг, 5 авторів і 3 категорій; фільтрує обидва варіанти за тією самою тестовою категорією, виконує один warm-up і 7 замірів, а виводить SQL кожного варіанта та медіану часу в мілісекундах. Наївний тестовий шлях виконує окреме завантаження авторів і категорії для кожної книги; production шлях — `List` з GORM Preload. Перевірка очікує 21 SELECT проти 4 SELECT: один запит для `books`, окремі пакетні запити для `book_authors` і `authors`, та один запит для `categories` (на тестовому наборі всі книги мають одну категорію). Ці числа є умовами тесту, а не результатом вимірювання в цьому середовищі.

Запуск порівняння на окремій PostgreSQL тестовій базі (команда виводить справжні SQL і час у `go test -v`):

```powershell
$env:TEST_DATABASE_URL = "postgres://<user>:<password>@localhost:5432/libratrack_test?sslmode=disable"
go test ./internal/repository -run '^TestPostgresBookListNPlusOneComparison$' -count=1 -v
```

#### Фактичний результат запуску

**Статус: PostgreSQL інтеграційні тести виконано на локальній `go_test_db`.** Тест legacy-міграції `up/down/up` пройшов; тест N+1 порівняння пройшов із 21 SELECT до оптимізації та 4 SELECT після `Preload`. API smoke test окремо не запускався. Нижче наведено фактичні counts, latency та приклади SQL з успішного запуску `TestPostgresBookListNPlusOneComparison`.

| Варіант | Фактичний SELECT count | Фактична latency (медіана, ms) | SQL log |
|---|---:|---:|---|
| До: наївне завантаження | 21 | 2.796 | Книги: 1; автори: 10; категорії: 10 |
| Після: GORM `Preload` | 4 | 1.042 | Книги: 1; `book_authors`: 1; автори: 1; категорії: 1 |

Фрагменти фактичного SQL з того самого запуску (наївні запити до авторів/категорій повторювалися по одному разу на кожну з 10 книг):

```text
BEFORE naive: SELECTs=21 first-run=2.764ms median=2.796ms
SELECT * FROM "books" WHERE books.category_id = 41 ORDER BY books.id ASC LIMIT 100
SELECT "authors"."id","authors"."full_name","authors"."bio","authors"."created_at","authors"."updated_at" FROM "authors" JOIN "book_authors" ON "book_authors"."author_id" = "authors"."id" AND "book_authors"."book_id" = 117
SELECT * FROM "categories" WHERE "categories"."id" = 41 ORDER BY "categories"."id" LIMIT 1

AFTER Preload: SELECTs=4 first-run=1.052ms median=1.042ms
SELECT * FROM "book_authors" WHERE "book_authors"."book_id" IN (117,118,119,120,121,122,123,124,125,126)
SELECT * FROM "authors" WHERE "authors"."id" IN (57,58,59,60,61)
SELECT * FROM "categories" WHERE "categories"."id" = 41
SELECT * FROM "books" WHERE books.category_id = 41 ORDER BY books.id ASC LIMIT 100
```

Час виміряно локально на тестовій PostgreSQL БД; це результат одного тестового запуску, а не узагальнена продуктивність.

Додаткові перевірки, виконані раніше без PostgreSQL: `go test -race ./...`, `go vet ./...`, `go build ./...`. Після налаштування `TEST_DATABASE_URL` повторно пройшов повний набір `go test ./... -count=1`. Інтеграційні тести PostgreSQL пропускаються, якщо ця змінна не задана. Див. [контрольні точки](checkpoints/README.md).

## 7. Здача

Репозиторій із коректним `VARIANT`, зеленим CI і короткою секцією в цьому README: як запустити, що реалізовано,
які рішення ви прийняли. Дедлайн і форма захисту: за вказівкою викладача.
