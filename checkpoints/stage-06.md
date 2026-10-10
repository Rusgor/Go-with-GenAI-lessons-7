# Етап 6 — PostgreSQL схема та міграції

## Мета
Підготувати LibraTrack до роботи з PostgreSQL через Compose та керовані SQL-міграції, не змінюючи сховище Lesson 7.

## Виконано
- Додано `docker-compose.yml`, `.env.example` і правило ігнорування локального `.env`.
- Додано основну послідовність `migrations/000001_init_schema` та `migrations/000002_seed_demo_data` із таблицями `categories`, `authors`, `books`, `book_authors`, `copies`, `users`.
- Схема має зовнішні ключі й індекси, `RESTRICT` для категорій/авторів, каскадне видалення зв’язків книги та копій, унікальні ISBN/інвентарні номери, CHECK для статусу копії та ролі користувача.
- Додано окрему послідовність у `migrations-homework/migrations/`, що демонструє перехід із legacy `books.author`/`books.category` та містить запитані `000002_normalize_books.up.sql` і `.down.sql`. У ній однакові версії не змішані з основною послідовністю.
- Початкові дані містять 10 книг, 5 авторів, 3 категорії та 20 копій.

## Перевірки
- SQL міграції не були застосовані: у локальному середовищі немає Docker/Podman, `psql` або `migrate`. Перевірки up/down/up і обмежень PostgreSQL залишаються незавершеними.
- Зміни збережені без редагування untracked `.github/workflows/theme8-check.yml` і `libratrack_lesson8_homework.md`.

## Статус
Файли й схема підготовлені; виконання на реальному PostgreSQL очікує доступного локального або CI середовища.
