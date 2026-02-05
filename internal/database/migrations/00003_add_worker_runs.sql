-- +goose Up
CREATE TYPE worker_run_status AS enum ('running', 'completed', 'failed');

CREATE TABLE worker_runs
    (
        id uuid
            PRIMARY KEY DEFAULT gen_random_uuid(),
        worker_name VARCHAR(64) NOT NULL,
        status worker_run_status NOT NULL DEFAULT 'running',
        started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        ended_at TIMESTAMP
    );

CREATE INDEX idx_worker_runs_worker_name ON worker_runs (worker_name);
CREATE INDEX idx_worker_runs_status ON worker_runs (status);
CREATE INDEX idx_worker_runs_started_at ON worker_runs (started_at);

-- +goose Down
DROP TRIGGER if EXISTS set_updated_at_worker_runs ON worker_runs;
DROP TABLE if EXISTS worker_runs;
DROP TYPE if EXISTS worker_run_status;
