# Описание
#### Проект выполнен для курса "[Микросервисная архитектура](https://otus.ru/lessons/microservice-architecture)"

## [Архитектура сервиса](doc/description.md)

## [Распределенные транзакции](doc/saga.md)

# Сервисы
Сами сервисы расположен в папке `services`, при запуске в ней команды `make` собираются docker-образы сервисов.
API всех сервисов можно посмотреть в папке `service/api`

# Инструкция по запуску

### Предварительная подготовка
1. Прописать в hosts домен `arch.homework` на ip кластера
2. При необходимости создать новый namespace и выбрать его, например:
```
kubectl create namespace arch-project && kubectl config set-context --current --namespace=arch-project
```
3. Установить Prometheus и Nginx при отсутствии:
```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add ingress-nginx 
helm repo update

helm install prom prometheus-community/kube-prometheus-stack -f external/prometheus.yaml --atomic
helm install nginx ingress-nginx/ingress-nginx -f external/nginx-ingress.yaml --atomic
``` 

### Установка приложения с помощью helm:
```
helm upgrade --install arch helm/hw-umbrella-chart
```

# Просмотр сообщений в очереди Rabbit MQ
Для просмотра сообщений в очереди событий Rabbit MQ Stream написана отдельная утилитка.  
Для ее сборки надо запустить команду `make` в папке `tools`.  
  
После сборки, можно запускать утилиту, передавая ей в качестве параметра желаемое количество последних сообщений. Например, для просмотра последних 10 сообщений в очереди нужно запустить команду
```
tools/bin/streamreader -count 10
```

# Тестирование
### Запуск тестов:
```
 newman run tests.postman_collection.json
```