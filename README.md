Тестовое задание. Создать API:
- запрос кода авторизации
- авторизация пользователя
- получение списка пользователей. Закрытый url, доступен через jwt token.
Результат авторизации пользователя: JWT токен, который даст доступ к получению списка пользователей, если пользователь существует обновляется дата последней авторизации, если пользователя нет - создается новая запись.  Полученный токен пользователя сохранить в БД.
