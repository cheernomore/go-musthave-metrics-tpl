CREATE TABLE IF NOT EXISTS metrics (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(10) NOT NULL CHECK (type IN ('counter', 'gauge')),
    delta BIGINT,
    value DOUBLE PRECISION,
    CONSTRAINT check_counter_delta CHECK (type != 'counter' OR delta IS NOT NULL),
    CONSTRAINT check_gauge_value CHECK (type != 'gauge' OR value IS NOT NULL)
);

CREATE INDEX idx_metrics_type ON metrics(type);
CREATE INDEX idx_metrics_name ON metrics(name);