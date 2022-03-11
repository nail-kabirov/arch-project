# Описание
#### Домашнее задание выполнено для курса "[Микросервисная архитектура](https://otus.ru/lessons/microservice-architecture)"

# Сервисы
Сами сервисы расположен в папке `services`, при запуске в ней команды `make` собираются docker-образы сервисов.
API всех сервисов можно посмотреть в папке `service/api`

# Распределенные транзакции
Для распределенных транзакций в проекте используются хореографические саги.
  
Для примера, рассмотрим операцию добавления новой ставки на лот:
1. Пользователь отправляет запрос `/lot/api/v1/lot/:lotId/bid` с суммой ставки в теле запроса и хедером `X-Request-ID` 
2. Сервис `Lot` предварительно проверяет, что переданный `X-Request-ID` еще не обрабатывался. Если же уже обрабатывался, то вернет ответ с HTTP-кодом 409
3. Сервис `Lot` отправляет синхронный запрос в сервис `Billing` на блокировку средств на ставку на счету пользователя. Если операция не прошла, то возвращает ошибку и прекращает выполнение запроса. В случае успеха сумма ставки на счету пользователя становится заблокированной
4. Сервис `Lot` запускает локальную транзакцию в которой проверяет все условия для добавления новой ставки:
   * Запрос с переданным `X-Request-ID` еще не обрабатывался (повторная проверка, но уже внутри транзакции) 
   * Лот активный, время окончания аукциона еще не наступило
   * Ставка больше или равна минимальной ставке для лота
   * Ставка больше последней ставки на этот лот (если ставки ранее были)
5. Если какое-то из условий пункта 4 не выполнено, то сага считается неуспешной. Сервис `Lot` отправляет событие `lot.bid_cancelled` для компенсации пункта 3. Сервис `Billing` обработает это событие и разблокирует средства на ставку на счету пользователя
6. Если все условия из пункта 4 выполнены ставка создается. Сага считается успешно выполненной

В случае успешного создания ставки в дальнейшем средства на счету могут либо разблокироваться (если ставку перебьет другой пользователь), либо списаться окончательно (после успешной отправки и получения лота).

# Инструкция по запуску

### Предварительная подготовка
1. Прописать в hosts домен `arch.homework` на ip кластера
2. При необходимости создать новый namespace и выбрать его, например:
```
kubectl create namespace arch-project && kubectl config set-context --current --namespace=arch-project
```
3. Установить Nginx при отсутствии (или включить addon в minikube - `minikube addons enable ingress`)

### Установка приложения с помощью helm:
```
helm upgrade --install arch helm/hw-umbrella-chart
```
# Тестирование
### Запуск тестов:
```
 newman run tests.postman_collection.json
```