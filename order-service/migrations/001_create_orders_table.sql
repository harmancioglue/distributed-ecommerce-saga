-- Orders tablosu
CREATE TABLE IF NOT EXISTS orders (
                                      id UUID PRIMARY KEY,
                                      customer_id UUID NOT NULL,
                                      items JSONB NOT NULL,
                                      failure_reason TEXT,
                                      total_amount DECIMAL(10,2) NOT NULL CHECK (total_amount >= 0),
    status VARCHAR(20) NOT NULL CHECK (status IN (
                                       'pending', 'processing', 'completed', 'cancelled', 'failed'
                                                 )),
    shipping_address JSONB NOT NULL,
    saga_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
    );

-- Performance indexleri
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_saga_id ON orders(saga_id) WHERE saga_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

CREATE INDEX IF NOT EXISTS idx_orders_processing_saga ON orders(saga_id, status)
    WHERE status IN ('processing') AND saga_id IS NOT NULL;