-- Saga instances table
CREATE TABLE IF NOT EXISTS saga_instances (
                                              id UUID PRIMARY KEY,
                                              order_id UUID NOT NULL UNIQUE,  -- each order can have one saga
                                              customer_id UUID NOT NULL,
                                              status VARCHAR(20) NOT NULL CHECK (status IN (
                                                                                 'started', 'in_progress', 'completed', 'failed', 'compensating', 'compensated'
                                                                                           )),
    current_step VARCHAR(50) NOT NULL,
    completed_steps JSONB NOT NULL DEFAULT '[]',
    failure_reason TEXT,
    context JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
                             );

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_saga_instances_order_id ON saga_instances(order_id);
CREATE INDEX IF NOT EXISTS idx_saga_instances_status ON saga_instances(status);
CREATE INDEX IF NOT EXISTS idx_saga_instances_created_at ON saga_instances(created_at);

-- For recovery to find in progress
CREATE INDEX IF NOT EXISTS idx_saga_instances_in_progress ON saga_instances(status, created_at)
    WHERE status IN ('started', 'in_progress', 'compensating');

-- Saga event log table (debugging ve monitoring)
CREATE TABLE IF NOT EXISTS saga_event_log (
                                              id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saga_id UUID NOT NULL REFERENCES saga_instances(id),
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    service_name VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    correlation_id UUID NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_saga_event_log_saga_id ON saga_event_log(saga_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_saga_event_log_correlation_id ON saga_event_log(correlation_id);