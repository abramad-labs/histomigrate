CREATE TABLE orders(
    id BIGSERIAL PRIMARY KEY,
    "number" VARCHAR(64) NOT NULL,
    account_ref BIGINT NOT NULL
);