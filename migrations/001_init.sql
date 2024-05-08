-- +goose Up
CREATE TABLE IF NOT EXISTS metric_counters (
    "name" VARCHAR (50) UNIQUE NOT NULL,
    "value" BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS metric_gauges (
    "name" VARCHAR (50) UNIQUE NOT NULL,
    "value" DOUBLE PRECISION NOT NULL DEFAULT 0
);


-- +goose Down
DROP TABLE metric_counters;
DROP TABLE metric_gauges;
