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

CREATE INDEX IF NOT EXISTS idx_products_stock ON products(stock, reserved_stock);
CREATE INDEX IF NOT EXISTS idx_reservations_order_id ON inventory_reservations(order_id);
CREATE INDEX IF NOT EXISTS idx_reservations_saga_id ON inventory_reservations(saga_id);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON inventory_reservations(status);
CREATE INDEX IF NOT EXISTS idx_reservations_expires ON inventory_reservations(expires_at) WHERE status = 'reserved';

ALTER TABLE products ADD CONSTRAINT chk_reserved_stock_limit CHECK (reserved_stock <= stock);

INSERT INTO products (id, name, price, stock) VALUES 
    ('550e8400-e29b-41d4-a716-446655440001', 'Product 1', 29.99, 100),
    ('550e8400-e29b-41d4-a716-446655440002', 'Product 2', 49.99, 50),
    ('550e8400-e29b-41d4-a716-446655440003', 'Product 3', 99.99, 25)
ON CONFLICT (id) DO NOTHING;
