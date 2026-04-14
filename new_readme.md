# Warehouse Management Project

## Обзор проекта

Этот проект состоит из двух микросервисов, реализующих упрощённую, но демонстративную систему управления складом и взаимодействия с пользователями через корпоративную прослойку.

- **Warehouse Management System (WMS)** — основной сервис, который принимает заказы, резервирует остатки, планирует сборку заказов в волнах и управляет состоянием заказа.
- **Bot** — сервис-обёртка, который принимает входящие сообщения от пользователей, отправляет уведомления пользователям и служит прокси между клиентом и WMS и его основа это телграмм бот и библиотека "github.com/go-telegram/bot".

Проект будет реализовывать типовые архитектурные паттерны, при этом как на уровне сервиса так и на уровне меджсервисного взаимодействия: по архитектуре - DDD, hexagonal architecture; по технологиям - gRPC, Kafka для асинхронных уведомлений, retry/circuit breaker и rate limiting + outbox для отказоустойчивости и postgresql как основное хранилище.

## Цель и границы

Проект не будет пытаться покрыть весь мир e-commerce. Он делает акцент на следующих бизнес-ценностях:

- приём и базовая обработка заказов
- резервирование складских остатков
- формирование и управление волнами комплектации
- уведомление пользователей о статусе заказа
- простая модель пользователей без сложных ролей и прав

## Структура системы

### 1. Warehouse Management System

WMS отвечает за:

- CRUD заказов
- резервирование остатков на складе
- обработку частичных резервов и backorder
- построение и запуск волнового планирования
- управление состоянием заказа через явные переходы состояний
- публикацию событий в Kafka при необходимости уведомления клиенту(т.е заказ сформированан)

Это ядро проекта. Внутри WMS сохраняются доменные сущности и реализуются бизнес-правила тут будем стараться домен сделать максимально близко в DDD тюе использовать domain service и сделать богатые доменные модели.

### 2. Bot / Frontend Messaging Service

Корпоративная прослойка принимает команды от пользователей и отправляет уведомления обратно.

- приём входящего запроса от клиента и отправка уведомлений с использваонием ("github.com/go-telegram/bot".)
- преобразование пользовательского запроса в gRPC вызовы к WMS делаем прото и генерим через protoc клиаент и сервер
- прием уведомлений от WMS через Kafka (inbox для Bot Service и outbox для WMS)
- retry backoff и circuit breaker(как grpc interceptor для mws) для внешних вызовов и доставки сообщений

Этот сервис делает взаимодействие пользователя со складом удобным и устойчивым.

## Доменные сущности

### Order

Заказ — центральная сущность(частично он уже реализован). Он содержит:

- уникальный идентификатор
- список позиций (SKU + количество)
- текущий статус
- временные метки создания и изменений

Переходы статусов должны быть контролируемыми, чтобы в системе сохранялась последовательность работы.

### OrderItem

Строка заказа описывает отдельную товарную позицию(тоже частино реализован):

- SKU
- количество

В более сложной системе сюда можно добавить цену, указания по упаковке, размер, вес.

### Reservation

Резерв — это привязка части доступного склада к конкретному заказу.

- резерв может быть полный или частичный
- при нехватке доступного количества часть товара уходит в backorder
- резерв должен быть транзакционным, чтобы избежать рассинхронизации остатков

### Wave

Волна — это пакет заказов, собранных для комплектации за один цикл.

- волновое планирование объединяет заказы по времени, маршруту, зоне или приоритету
- цель — уменьшить время перемещения и ускорить сборку
- волна формируется по порогу количества заказов или по интервалу времени

### User

Пользовательская модель в проекте будет простой: сейчас любой клиент который зарегался в боте может добавлять заказы  и они к нему будут относиться и ему по ним будут уведомление приходить

## Основные юзкейсы

### 1. Регистрация и управление пользователями

- пользователь регистрируется в системе
- система сохраняет chatID и он имя еще введет пускай

### 2. CRUD для заказов

Основные операции над заказами:

- Create — принять новый заказ с позициями
- Read — получить заказ по `id`, посмотреть текущее состояние и резервы
- Update — изменить заказ, если он ещё не находится в стадии `in_wave` или позже
- Delete — удалить заказ до начала процесса комплектации

Этот набор операций покрывает базовый жизненный цикл заказа и достаточен для учебного проекта.

### 3. Планирование сборки заказов

Planning system отвечает за упаковку заказов в волны и за создание задач для исполнения.

- определение момента запуска волны (порог, время, очередность)
- выбор заказов для включения в волну
- распределение заказов по зонам или группам
- создание задач сборки/упаковки

Понятие волны в проекте можно реализовать как:

- если число заказов достигает `limit`
- или если прошло `interval` времени
- или если накопился приоритетный заказ

Это позволяет демонстрировать как синхронную, так и асинхронную логику.

### 4. Уведомления пользователю и от пользователя к системе

Сервис бота работает как мост между пользователем и WMS.

- пользователь отправляет команду или сообщение боту
- бот переводит это в gRPC вызов в WMS
- WMS отвечает и может инициировать уведомление в Kafka
- бот получает уведомление через входную очередь (inbox) и доставляет его пользователю

В примере проекта реализуется двухсторонняя коммуникация:

- `user -> bot -> WMS`
- `WMS -> Kafka outbox -> Bot inbox -> user`

## Коммуникация и протоколы

### gRPC как основной метод взаимодействия

Для обмена между сервисами и для основных операций проекта предлагается использовать gRPC. Это позволит показать:

- чётко типизированный контракт между сервисами
- простой механизм для удалённого вызова бизнес-операций
- возможность генерировать клиентские библиотеки

Примерный протокол gRPC для уведомлений и CRUD заказов:

```proto
syntax = "proto3";
package warehouse;

service WarehouseService {
  rpc CreateOrder(CreateOrderRequest) returns (OrderResponse);
  rpc GetOrder(GetOrderRequest) returns (OrderResponse);
  rpc UpdateOrder(UpdateOrderRequest) returns (OrderResponse);
  rpc DeleteOrder(DeleteOrderRequest) returns (DeleteOrderResponse);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
}

service NotificationService {
  rpc SendNotification(NotificationRequest) returns (NotificationResponse);
}

message CreateOrderRequest {
  string user_id = 1;
  repeated OrderItem items = 2;
}

message OrderItem {
  string sku = 1;
  int32 quantity = 2;
}

message OrderResponse {
  string order_id = 1;
  string status = 2;
  repeated OrderItem items = 3;
  string error = 4;
}

message NotificationRequest {
  string id = 1;
  string user_id = 2;
  string order_id = 3;
  string channel = 4;
  map<string, string> data = 5;
}
```

### Kafka: outbox / inbox

Для уведомлений предлагается использовать подход `outbox` / `inbox`:

- WMS пишет событие уведомления в свой `outbox` (локально в БД или в Kafka) после успешной бизнес-операции.
- Bot Service читает это событие из `inbox` и формирует сообщение для пользователя.

Такой паттерн позволяет гарантировать, что уведомление не потеряется и доставляется хотя бы один раз. Он хорошо сочетается с паттернами retry, дедупликации и eventual consistency.

## Нефункциональные требования

- DDD + Hexagonal architecture. Бизнес-логика должна быть отделена от инфраструктуры.
- Транзакции для резервирования остатков.
- Ограничение частоты запросов (`rate limiter`) на стороне WMS, чтобы безопасно обрабатывать входящие вызовы.
- `retry` и `circuit breaker` на стороне Bot Service для внешних API (например, Telegram) и граничных сервисов.
- Структурированное логирование и метрики.
- Поддержка контейнеризации.

## Что уже есть в доменной модели

package domain

import (
 "errors"
 "slices"
 "time"
)

var (
 ErrInvalidOrderItems = errors.New("order must have at least one item")
 ErrIllegalStatusStep = errors.New("illegal status transition")
)

type OrderStatus string

const (
 StatusNew               OrderStatus = "new"
 StatusReserving         OrderStatus = "reserving"
 StatusReserved          OrderStatus = "reserved"
 StatusPartiallyReserved OrderStatus = "partially_reserved"
 StatusInWave            OrderStatus = "in_wave"
 StatusShipped           OrderStatus = "shipped"
 StatusCancelled         OrderStatus = "cancelled"
)

type OrderItem struct {
 SKU      string
 Quantity int
}

type Order struct {
 id        int64
 status    OrderStatus
 items     []OrderItem
 createdAt time.Time
}

func NewOrder(id int64, items []OrderItem) (*Order, error) {
 if len(items) == 0 {
  return nil, ErrInvalidOrderItems
 }

 for _, item := range items {
  if item.Quantity <= 0 {
   return nil, errors.New("item quantity must be positive")
  }
 }

 return &Order{
  id:        id,
  status:    StatusNew,
  items:     items,
  createdAt: time.Now(),
 }, nil
}

func (o *Order) ID() int64 {
 return o.id
}

func (o *Order) Status() OrderStatus {
 return o.status
}

func (o *Order) Items() []OrderItem {
 return slices.Clone(o.items)
}

func (o *Order) TransitionTo(next OrderStatus) error {
 allowed := false

 // todo: all statuses
 switch o.status {
 case StatusNew:
  allowed = (next == StatusReserving || next == StatusCancelled)
 case StatusReserving:
  allowed = (next == StatusReserved || next == StatusPartiallyReserved || next == StatusCancelled)
 case StatusReserved, StatusPartiallyReserved:
  allowed = (next == StatusInWave || next == StatusCancelled)
 }

 if !allowed {
  return ErrIllegalStatusStep
 }

 o.status = next

 return nil
}

package domain

type ReservationResult struct {
 ReservedQty  int
 BackorderQty int
}

type Stock struct {
 sku           string
 totalQuantity int
 reservedQty   int
}

func NewStock(sku string, total int) *Stock {
 return &Stock{
  sku:           sku,
  totalQuantity: total,
 }
}

func (s *Stock) Reserve(requestedQty int) ReservationResult {
 available := max(s.totalQuantity-s.reservedQty, 0)
 toReserve := min(requestedQty, available)

 s.reservedQty += toReserve

 return ReservationResult{
  ReservedQty:  toReserve,
  BackorderQty: requestedQty - toReserve,
 }
}

func (s *Stock) Release(qty int) {
 s.reservedQty -= qty
 if s.reservedQty < 0 {
  s.reservedQty = 0
 }
}

package domain

import "errors"

type WaveStatus string

const (
 WaveStatusOpen      WaveStatus = "open"       // Сбор заказов
 WaveStatusInProcess WaveStatus = "in_process" // Задачи созданы
 WaveStatusCompleted WaveStatus = "completed"
)

type Wave struct {
 id        int64
 orders    []int64
 status    WaveStatus
 maxOrders int
}

func NewWave(id int64, maxOrders int) *Wave {
 return &Wave{
  id:        id,
  status:    WaveStatusOpen,
  maxOrders: maxOrders,
  orders:    make([]int64, 0),
 }
}

func (w *Wave) AddOrder(orderID int64) error {
 if w.status != WaveStatusOpen {
  return errors.New("cannot add orders to non-open wave")
 }

 if len(w.orders) >= w.maxOrders {
  return errors.New("wave is full")
 }

 w.orders = append(w.orders, orderID)

 return nil
}

func (w *Wave) Orders() []int64 {
 cp := make([]int64, len(w.orders))
 copy(cp, w.orders)

 return cp
}

## Предложенная логика планирования

Планирование будет строиться вокруг следующих идей:

- заказы проходят стадию резервирования, затем переходят в состояние `reserved` или `partially_reserved`;
- когда накопилось достаточное число заказов или прошло заданное время, система формирует волну;
- в волну попадают заказы, которые готовы к сборке;
- волна создаёт задачи для бригад/роботов/сборщиков(нам без разницы улоснво это);
- после выполнения задач статус заказа переводится в `packed` и затем `shipped`.

Юзкейсы планирования:

- запуск создания волны по порогу количества заказов;
- запуск волны по таймеру для повышения регулярности;
- эскалация при наличии заказов с высоким приоритетом;
- просмотр статуса волны и связанных задач;
- завершение волны и перевод заказов в следующую стадию.
- в общем алгоритм отчасти похож на алгоритм garbage collector golang

## Итог

Проект должен оставаться компактным, но показывать несколько ключевых архитектурных идей:

- разделение ответственности между сервисами
- использование gRPC для бизнес-интерфейсов
- асинхронное уведомление через Kafka
- паттерны надежности и отказоустойчивости: retry, circuit breaker, rate limiting
- понятная доменная модель заказа максимально приближенная в DDD идиоматичной модели

Такой подход позволит реализовать не «супербольшой» сервис, но достаточно интересное и структурированное приложение для курсового проекта.
