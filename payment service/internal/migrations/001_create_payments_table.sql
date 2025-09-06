CREATE TABLE IF NOT EXISTS payments (
                                        id UUID PRIMARY KEY,
                                        order_id UUID NOT NULL,
                                        customer_id UUID NOT NULL,
                                        saga_id UUID NOT NULL,
                                        amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN (
                                       'pending', 'completed', 'failed', 'refunded'
                                                 )),
    transaction_id VARCHAR(255),
    external_ref VARCHAR(255),
    failure_reason TEXT,
    refunded_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00 CHECK (refunded_amount >= 0),
    refund_reference VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE
                             );

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_saga_id ON payments(saga_id);
CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments(customer_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_transaction_id ON payments(transaction_id) WHERE transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at);

-- Business logic constraints
CREATE INDEX IF NOT EXISTS idx_payments_pending ON payments(saga_id, status)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_payments_refundable ON payments(order_id, status, refunded_amount)
    WHERE status = 'completed' AND refunded_amount < amount;

-- Ensure refunded amount doesn't exceed payment amount
ALTER TABLE payments ADD CONSTRAINT chk_refunded_amount_limit
    CHECK (refunded_amount <= amount);

-- Unique constraint per order (bir order için bir payment)
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_unique_order
    ON payments(order_id);