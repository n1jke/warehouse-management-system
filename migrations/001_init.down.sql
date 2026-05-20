BEGIN;

DROP TABLE outbox;
DROP TABLE reservations;
DROP TABLE stocks;
DROP TABLE wave_orders;
DROP TABLE waves;
DROP TABLE order_items;
DROP TABLE orders;
DROP TABLE users;

COMMIT;
