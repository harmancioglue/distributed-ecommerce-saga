-- Initialize databases for all microservices
-- This script runs automatically when PostgreSQL container starts

-- Create databases for each service
CREATE DATABASE order_db;
CREATE DATABASE payment_db;
CREATE DATABASE inventory_db;
CREATE DATABASE shipping_db;
CREATE DATABASE notification_db;
CREATE DATABASE orchestrator_db;

-- Grant permissions to saga_user for all databases
GRANT ALL PRIVILEGES ON DATABASE order_db TO saga_user;
GRANT ALL PRIVILEGES ON DATABASE payment_db TO saga_user;
GRANT ALL PRIVILEGES ON DATABASE inventory_db TO saga_user;
GRANT ALL PRIVILEGES ON DATABASE shipping_db TO saga_user;
GRANT ALL PRIVILEGES ON DATABASE notification_db TO saga_user;
GRANT ALL PRIVILEGES ON DATABASE orchestrator_db TO saga_user;

-- Switch to each database and run initialization
\c order_db;
-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    items JSONB NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL CHECK (total_amount >= 0),
    status VARCHAR(20) NOT NULL CHECK (status IN (
        'pending', 'processing', 'completed', 'cancelled', 'failed'
    )),
    shipping_address JSONB NOT NULL,
    saga_id UUID,
    failure_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_saga_id ON orders(saga_id) WHERE saga_id IS NOT NULL;

\c payment_db;
-- Payments table
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE,
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
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
ALTER TABLE payments ADD CONSTRAINT chk_refunded_amount_limit CHECK (refunded_amount <= amount);

\c inventory_db;
-- Products and inventory reservations
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    reserved_stock INTEGER NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inventory_reservations (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    product_id UUID NOT NULL REFERENCES products(id),
    saga_id UUID NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status VARCHAR(20) NOT NULL CHECK (status IN (
        'available', 'reserved', 'released', 'sold'
    )),
    reserved_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reservations_saga_id ON inventory_reservations(saga_id);
ALTER TABLE products ADD CONSTRAINT chk_reserved_stock_limit CHECK (reserved_stock <= stock);

-- Insert sample products
INSERT INTO products (id, name, price, stock) VALUES 
    ('550e8400-e29b-41d4-a716-446655440001', 'Laptop Pro 15', 1299.99, 50),
    ('550e8400-e29b-41d4-a716-446655440002', 'Wireless Mouse', 49.99, 100),
    ('550e8400-e29b-41d4-a716-446655440003', 'USB-C Hub', 79.99, 75),
    ('550e8400-e29b-41d4-a716-446655440004', 'Mechanical Keyboard', 159.99, 30),
    ('550e8400-e29b-41d4-a716-446655440005', 'Monitor 24 inch', 299.99, 25)
ON CONFLICT (id) DO NOTHING;

\c shipping_db;
-- Shipments table
CREATE TABLE IF NOT EXISTS shipments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE,
    customer_id UUID NOT NULL,
    saga_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN (
        'pending', 'preparing', 'shipped', 'delivered', 'cancelled'
    )),
    tracking_id VARCHAR(255) NOT NULL,
    address JSONB NOT NULL,
    failure_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_shipments_order_id ON shipments(order_id);
CREATE INDEX IF NOT EXISTS idx_shipments_saga_id ON shipments(saga_id);
CREATE INDEX IF NOT EXISTS idx_shipments_tracking_id ON shipments(tracking_id);

\c notification_db;
-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    saga_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('email', 'sms', 'push')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'sent', 'failed')),
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_notifications_order_id ON notifications(order_id);
CREATE INDEX IF NOT EXISTS idx_notifications_saga_id ON notifications(saga_id);

\c orchestrator_db;
-- Saga instances and event log
CREATE TABLE IF NOT EXISTS saga_instances (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE,
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

CREATE TABLE IF NOT EXISTS saga_event_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saga_id UUID NOT NULL REFERENCES saga_instances(id),
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    service_name VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    correlation_id UUID NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_saga_instances_order_id ON saga_instances(order_id);
CREATE INDEX IF NOT EXISTS idx_saga_instances_status ON saga_instances(status);
CREATE INDEX IF NOT EXISTS idx_saga_event_log_saga_id ON saga_event_log(saga_id, timestamp);
