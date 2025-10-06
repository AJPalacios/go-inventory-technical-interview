-- Datos de prueba para el sistema de inventario

-- Limpiar datos existentes (opcional)
DELETE FROM reservations;
DELETE FROM inventory_items;
DELETE FROM products;
DELETE FROM idempotency_keys;

-- Insertar productos de prueba
INSERT INTO products (id, name, description) VALUES
('prod-001', 'Laptop HP Pavilion 15', 'Laptop HP con procesador Intel Core i5, 8GB RAM, 256GB SSD'),
('prod-002', 'Mouse Logitech MX Master 3', 'Mouse inalámbrico ergonómico para productividad'),
('prod-003', 'Teclado Mecánico Keychron K2', 'Teclado mecánico 75% con switches Gateron Brown'),
('prod-004', 'Monitor Dell 27" 4K', 'Monitor 4K UHD de 27 pulgadas, IPS, 60Hz'),
('prod-005', 'Webcam Logitech C920', 'Cámara web Full HD 1080p con micrófono estéreo'),
('prod-006', 'Audífonos Sony WH-1000XM4', 'Audífonos con cancelación de ruido activa'),
('prod-007', 'SSD Samsung 1TB', 'Unidad de estado sólido NVMe 1TB, 3500 MB/s'),
('prod-008', 'RAM Corsair 16GB DDR4', 'Memoria RAM DDR4 3200MHz, 2x8GB'),
('prod-009', 'Hub USB-C Anker', 'Hub 7 en 1 con HDMI, USB 3.0 y lector SD'),
('prod-010', 'Cable HDMI 4K 2m', 'Cable HDMI 2.1 certificado, soporte 4K@120Hz');

-- Insertar inventario inicial
INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version) VALUES
('inv-001', 'prod-001', 25, 5, 1),
('inv-002', 'prod-002', 150, 10, 1),
('inv-003', 'prod-003', 80, 15, 1),
('inv-004', 'prod-004', 30, 0, 1),
('inv-005', 'prod-005', 45, 5, 1),
('inv-006', 'prod-006', 60, 10, 1),
('inv-007', 'prod-007', 100, 0, 1),
('inv-008', 'prod-008', 200, 20, 1),
('inv-009', 'prod-009', 75, 5, 1),
('inv-010', 'prod-010', 300, 0, 1);

-- Insertar algunas reservas de ejemplo
INSERT INTO reservations (id, request_id, product_id, quantity, status, expires_at) VALUES
('res-001', 'req-001', 'prod-001', 2, 'confirmed', datetime('now', '+7 days')),
('res-002', 'req-002', 'prod-002', 5, 'confirmed', datetime('now', '+5 days')),
('res-003', 'req-003', 'prod-003', 3, 'pending', datetime('now', '+2 days')),
('res-004', 'req-004', 'prod-006', 10, 'confirmed', datetime('now', '+10 days')),
('res-005', 'req-005', 'prod-008', 15, 'pending', datetime('now', '+3 days'));

-- Insertar claves de idempotencia de ejemplo
INSERT INTO idempotency_keys (request_id, operation_type, response_data, expires_at) VALUES
('req-001', 'reserve', '{"reservation_id":"res-001","status":"confirmed"}', datetime('now', '+1 day')),
('req-002', 'reserve', '{"reservation_id":"res-002","status":"confirmed"}', datetime('now', '+1 day'));

-- Verificar datos insertados
SELECT 'Products:' as table_name, COUNT(*) as count FROM products
UNION ALL
SELECT 'Inventory Items:', COUNT(*) FROM inventory_items
UNION ALL
SELECT 'Reservations:', COUNT(*) FROM reservations
UNION ALL
SELECT 'Idempotency Keys:', COUNT(*) FROM idempotency_keys;
