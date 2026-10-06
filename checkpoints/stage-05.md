# Звіт-контрольна точка: етап 5

**Тема:** CORS і конкурентність  
**Варіант:** `devices`  
**Статус:** виконано  
**Дата:** 2026-10-06

## Мета

Додати CORS до всіх HTTP-відповідей, обробити preflight `OPTIONS` і підтвердити безпечну конкурентну роботу API.

## Що реалізовано

- Додано middleware `CORS` у [internal/middleware/cors.go](../internal/middleware/cors.go). Він встановлює `Access-Control-Allow-Origin: *` та заголовки дозволених методів і запитних заголовків перед передаванням запиту далі.
- Для кожного `OPTIONS` middleware повертає `204 No Content`; `Access-Control-Allow-Methods` містить `GET, POST, PUT, DELETE, OPTIONS`, а `Access-Control-Allow-Headers` дозволяє `Content-Type`.
- У [internal/app/app.go](../internal/app/app.go) middleware обгортає весь `ServeMux`, тому CORS-заголовки застосовуються до health, API та невідомих маршрутів.
- Сховище й CRUD-логіку змінювати не знадобилося: воно вже використовує `sync.RWMutex`, повертає копії ресурсів і створюється окремо для кожного `NewRouter()`.

## Перевірки

Виконано:

```text
gofmt -w internal/middleware/cors.go internal/app/app.go
go test ./internal/app -run TestStage5 -v
go test ./...
go test -race ./...
go vet ./...
```

**Результат:** усі Stage 5 тести пройшли; повний набір тестів проєкту пройшов; `go test -race ./...` завершився без повідомлень про гонки; `go vet ./...` завершився без зауважень.

Тести етапу підтвердили `204` і потрібні CORS-заголовки для preflight, наявність allow-origin на звичайних відповідях, унікальні ID при 50 паралельних створеннях та успішні паралельні GET/PUT.

## Підсумок

Етап 5 завершено. Реалізація CORS застосована зовнішньою middleware-обгорткою; конкурентна безпека підтверджена повним запуском race detector.
