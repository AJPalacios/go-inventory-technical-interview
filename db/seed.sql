-- Datos de prueba para el sistema de inventario

-- Limpiar datos existentes (opcional)
DELETE FROM reservations;
DELETE FROM inventory_items;
DELETE FROM products;
DELETE FROM idempotency_keys;

-- Insertar productos de prueba (UUIDs válidos)
INSERT INTO products (id, name, description) VALUES
('2d70d1dc-cd3a-4f40-afb0-52e16445bf36', 'Laptop HP Pavilion 15', 'Laptop HP con procesador Intel Core i5, 8GB RAM, 256GB SSD'),
('2da3b8d3-69f1-46e6-a068-874532d5126a', 'Mouse Logitech MX Master 3', 'Mouse inalámbrico ergonómico para productividad'),
('e08e3e7e-9126-49e4-9caf-63885a07bd78', 'Teclado Mecánico Keychron K2', 'Teclado mecánico 75% con switches Gateron Brown'),
('fc39adf6-784c-43f3-bb0d-9ed79613dd21', 'Monitor Dell 27" 4K', 'Monitor 4K UHD de 27 pulgadas, IPS, 60Hz'),
('cf43ddc3-c4da-4a98-b011-67b33223fae1', 'Webcam Logitech C920', 'Cámara web Full HD 1080p con micrófono estéreo'),
('47569eb2-fe19-43cb-929d-aedfd59dc199', 'Audífonos Sony WH-1000XM4', 'Audífonos con cancelación de ruido activa'),
('f7d85ff3-6dbf-4ee8-bd61-54453610e441', 'SSD Samsung 1TB', 'Unidad de estado sólido NVMe 1TB, 3500 MB/s'),
('834004f0-f683-4e96-ae6b-bb6673869d24', 'RAM Corsair 16GB DDR4', 'Memoria RAM DDR4 3200MHz, 2x8GB'),
('cbb6a942-8687-4dd0-85ba-82f102f25ce1', 'Hub USB-C Anker', 'Hub 7 en 1 con HDMI, USB 3.0 y lector SD'),
('00907a59-5b4b-4432-8c49-e8bca4683799', 'Cable HDMI 4K 2m', 'Cable HDMI 2.1 certificado, soporte 4K@120Hz');

-- Insertar inventario inicial (UUIDs válidos)
INSERT INTO inventory_items (id, product_id, available_stock, reserved_stock, version) VALUES
('b882363a-9997-4b14-8c27-7bbd1fd82bb1', '2d70d1dc-cd3a-4f40-afb0-52e16445bf36', 25, 5, 1),
('65c09db6-076f-4de0-b8f4-828c5baf939b', '2da3b8d3-69f1-46e6-a068-874532d5126a', 150, 10, 1),
('43a9b325-5a08-48ed-8df4-e244487c77ec', 'e08e3e7e-9126-49e4-9caf-63885a07bd78', 80, 15, 1),
('99824463-4fe4-4105-9843-ccbe35846579', 'fc39adf6-784c-43f3-bb0d-9ed79613dd21', 30, 0, 1),
('bce19da1-f170-486a-8ddb-6921b18e6e8c', 'cf43ddc3-c4da-4a98-b011-67b33223fae1', 45, 5, 1),
('11366505-f9b7-489a-9da2-f05583cb1365', '47569eb2-fe19-43cb-929d-aedfd59dc199', 60, 10, 1),
('ac30fbf8-7344-4e55-9d50-6aca6470321f', 'f7d85ff3-6dbf-4ee8-bd61-54453610e441', 100, 0, 1),
('9eac7ebd-baf2-4c18-8885-4564a1556e06', '834004f0-f683-4e96-ae6b-bb6673869d24', 200, 20, 1),
('7c9f8710-2985-4542-a16a-a517fa3e1616', 'cbb6a942-8687-4dd0-85ba-82f102f25ce1', 75, 5, 1),
('82679685-6c67-4156-8c0e-25b564cb1f3e', '00907a59-5b4b-4432-8c49-e8bca4683799', 300, 0, 1);

-- Insertar algunas reservas de ejemplo (UUIDs válidos)
INSERT INTO reservations (id, request_id, product_id, quantity, status, expires_at) VALUES
('11171f8d-a6a4-42d3-9ab1-c6d3d829c83e', 'req-001', '2d70d1dc-cd3a-4f40-afb0-52e16445bf36', 2, 'confirmed', datetime('now', '+7 days')),
('1e30b07b-03cc-4b8d-9324-e69a316f0d5e', 'req-002', '2da3b8d3-69f1-46e6-a068-874532d5126a', 5, 'confirmed', datetime('now', '+5 days')),
('b97bdd7a-29bb-498a-8dcc-a523ea4cedd0', 'req-003', 'e08e3e7e-9126-49e4-9caf-63885a07bd78', 3, 'pending', datetime('now', '+2 days')),
('846c4180-de70-410f-bc4b-4d287b123f2f', 'req-004', '47569eb2-fe19-43cb-929d-aedfd59dc199', 10, 'confirmed', datetime('now', '+10 days')),
('ef9a3dc8-860f-4374-ab3c-4317f9f30d8c', 'req-005', '834004f0-f683-4e96-ae6b-bb6673869d24', 15, 'pending', datetime('now', '+3 days'));

-- Insertar claves de idempotencia de ejemplo (UUIDs válidos)
INSERT INTO idempotency_keys (request_id, operation_type, response_data, expires_at) VALUES
('req-001', 'reserve', '{"reservation_id":"11171f8d-a6a4-42d3-9ab1-c6d3d829c83e","status":"confirmed"}', datetime('now', '+1 day')),
('req-002', 'reserve', '{"reservation_id":"1e30b07b-03cc-4b8d-9324-e69a316f0d5e","status":"confirmed"}', datetime('now', '+1 day'));

-- Verificar datos insertados
SELECT 'Products:' as table_name, COUNT(*) as count FROM products
UNION ALL
SELECT 'Inventory Items:', COUNT(*) FROM inventory_items
UNION ALL
SELECT 'Reservations:', COUNT(*) FROM reservations
UNION ALL
SELECT 'Idempotency Keys:', COUNT(*) FROM idempotency_keys;
