# Етап 9 — SECURITY: автентифікація, RBAC та аудит

## Мета
Захистити LibraTrack API паролями bcrypt, JWT із 24-годинним строком дії та ролями reader/librarian/admin; задокументувати матрицю доступу й провести triage-аудит.

## Реалізовано
- Додані `000003_add_auth_fields` up/down: поле `password_hash`, перехід `member` → `reader`, нове обмеження ролей та зворотне перетворення під час rollback.
- Додана `000004_create_loans` up/down зі зв'язками користувач/копія, індексами та унікальним partial index для однієї активної позики на копію.
- Реєстрація та логін використовують окремі DTO. Пароль хешується bcrypt `DefaultCost`; роль під час публічної реєстрації завжди reader; довжина пароля обмежена 8–72 байтами. Невідомий email і неправильний пароль повертають ідентичний HTTP 401 `invalid credentials`.
- JWT видається тільки з `JWT_SECRET`, підписується HS256, містить `user_id`, `role`, `iat`, `exp` (24 години). Секрет коротший за 32 байти відхиляється; entrypoint-и LibraTrack не стартують без налаштованого секрету.
- Додані `AuthMiddleware` та `RequireRole` з явними `AbortWithStatusJSON`; маршрутні групи розділені на public, protected, staff та admin. GET каталогу публічний; зміни каталогу й операції з позиками — staff; перегляд власних позик — authenticated; список користувачів і зміна ролі — admin.
- Loan checkout/return атомарно оновлює позики та статус копії. GET власних позик фільтрує за `user_id` із перевірених JWT claims.
- `PasswordHash` має `json:"-"`; user list вибирає тільки публічні колонки. Логер Gin не читає та не журналює тіло запиту.
- Додані дизайн матриці доступу (`security_design.txt`), збережені тексти промптів (`prompts.txt`) та triage-аудит (`security_audit.md`). Публічна реєстрація не дозволяє створити admin: первинного admin слід призначати оператором БД через захищений привілейований канал після реєстрації облікового запису.

## Перевірки та фактичний результат
- `go test -v ./...` — пройшов; HTTP-тести перевіряють реєстрацію, дубль email (422), успішний JWT login (200), однакову помилку для обох типів невдалого login (401), відсутній/прострочений token (401), reader → POST books (403), librarian/admin → POST books (201) та admin-only доступ.
- PostgreSQL доступний через `TEST_DATABASE_URL`; `go test -v ./...` пройшов, включно з міграціями. Додатково `TestPostgresLoanCheckoutReturnAndReaderDefault` пройшов: перевірено reader default, checkout, власний список, конфлікт повторної видачі, повернення та відновлення статусу копії.
- `go test -race ./...` — пройшов.
- `go vet ./...` — пройшов.
- `go build ./...` — пройшов.
- `git diff --check` — пройшов.

## Результат
Основні auth/RBAC вимоги етапу реалізовані та перевірені; повний журнал triage — у [security_audit.md](../security_audit.md), матриця — у [security_design.txt](../security_design.txt), промпти — у [prompts.txt](../prompts.txt).
