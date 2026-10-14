# LibraTrack — security audit

Дата: 2026-10-14
Обсяг: auth, JWT/RBAC middleware, catalog handlers, user/loan endpoints, repository query paths. Результат — ручний triage security-review кандидатів, перевірка коду та HTTP/PostgreSQL тестами.

## Знахідки та triage

| # | Потенційна знахідка | Серйозність | Статус | Перевірка / виправлення |
|---|---|---|---|---|
| 1 | Мутації каталогу без авторизації могли дозволяти будь-якому мережевому клієнту створювати/змінювати/видаляти записи. | High | Fixed Now | `internal/app/library.go:39-47`: усі catalog write routes розміщені в staff-групі після `AuthMiddleware` та `RequireRole(librarian, admin)`; public GET лишилися окремо. Перевірено 401/403/201 у `internal/handler/security_test.go`. |
| 2 | Публічна реєстрація з прив'язкою JSON до User могла б дозволити клієнту призначити собі admin/staff роль (mass assignment). | High | Fixed Now | `internal/handler/auth_handler.go:27-39,57`: окремий `RegisterRequest`, роль задається сервером як `reader`; невідоме поле `role` не використовується. Тест реєструє запит із `role: admin` і підтверджує, що відповідь містить `reader`. |
| 3 | Витік password hash через JSON-відповіді після додавання облікового поля. | High | Fixed Now | `internal/model/user.go:15`: `PasswordHash` має `json:"-"`; список користувачів вибирає лише безпечні поля (`internal/repository/user_repository.go`). Тест реєстрації перевіряє відсутність `password_hash` у JSON. |
| 4 | Різні помилки login для невідомого email і неправильного пароля дали б змогу перераховувати користувачів. | Medium | Fixed Now | `internal/handler/auth_handler.go:69-78`: обидва випадки повертають однакові HTTP 401 та `invalid credentials`; перевірено тестом з порівнянням однакових тіл відповідей. |
| 5 | Відсутній Abort у перевірках middleware міг би дозволити виконання захищеного handler після відмови. | High | Fixed Now | `internal/middleware/auth.go:48-63` і `internal/middleware/rbac.go:9-25`: усі відмови завершують ланцюжок через `AbortWithStatusJSON`; тести підтверджують 401 без/з простроченим токеном і 403 для reader. |
| 6 | SQL injection через фільтр каталогу. | Medium | False Positive | Фільтри передаються як параметризовані значення в GORM (`internal/repository/book_repository.go:114-119`); ID попередньо розбирається як число у handler. Ознак конкатенації SQL із введенням клієнта не виявлено. |
| 7 | Логування пароля в тілі запиту. | Medium | False Positive | `gin.Logger()` у `internal/app/library.go:17` журналює HTTP-події, але middleware не читає і не виводить request body. Перевірено код логування та login handler; пароль не додається до лог-полів. |

## Додаткові перевірки

- Реєстрація використовує `bcrypt.GenerateFromPassword` з `bcrypt.DefaultCost`; логін — `bcrypt.CompareHashAndPassword`.
- JWT створюється з HS256, `user_id`, `role`, `iat`, `exp` на 24 години. Секрет береться тільки з `JWT_SECRET`; значення коротше 32 байтів блокується, fallback-секрету немає.
- Auth middleware перевіряє Bearer формат, HS256, підпис, exp, user_id та дозволену роль перед записом claims у Gin context.
- `GET /api/v1/me/loans` шукає позики за `user_id` із валідованого токена, не з параметра запиту.
- Loan checkout/return змінюють статус копії та запис позики в транзакції; активна позика на копію унікальна на рівні PostgreSQL partial index.
- Bootstrap першого admin виконується оператором БД після реєстрації першого користувача; публічна реєстрація не надає admin роль.
