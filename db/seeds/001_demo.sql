-- Development/demo data only. Never run this seed in production.
-- Login: tenant `nusantara-logistics`, email `admin@fleet.local`, password `Admin123!`.
-- The password below is an Argon2id v=19 hash (m=65536, t=3, p=4); plaintext is never stored.

BEGIN;

INSERT INTO tenants (id, name, slug, timezone, is_active)
VALUES ('10000000-0000-4000-8000-000000000001', 'Nusantara Logistics', 'nusantara-logistics', 'Asia/Jakarta', true)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    timezone = EXCLUDED.timezone,
    is_active = EXCLUDED.is_active;

INSERT INTO users (id, tenant_id, email, password_hash, full_name, status)
SELECT
    '20000000-0000-4000-8000-000000000001',
    t.id,
    'admin@fleet.local',
    '$argon2id$v=19$m=65536,t=3,p=4$ZmxlZXQtZGV2LXNlZWQhIQ$C2RLxCAz4oX2OJaJ9Se+7xjRYx7nnQm42JULOKOaa3w',
    'Nusantara Fleet Administrator',
    'active'
FROM tenants t
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, email) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    full_name = EXCLUDED.full_name,
    status = EXCLUDED.status,
    deleted_at = NULL;

INSERT INTO roles (id, tenant_id, name, description, is_system)
SELECT v.id, t.id, v.name, v.description, true
FROM tenants t
CROSS JOIN (VALUES
    ('30000000-0000-4000-8000-000000000001'::uuid, 'fleet_admin'::citext, 'Full tenant administration'),
    ('30000000-0000-4000-8000-000000000002'::uuid, 'dispatcher'::citext, 'Fleet operations and geofence management'),
    ('30000000-0000-4000-8000-000000000003'::uuid, 'viewer'::citext, 'Read-only fleet visibility')
) AS v(id, name, description)
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, name) DO UPDATE
SET description = EXCLUDED.description,
    is_system = EXCLUDED.is_system;

INSERT INTO permissions (id, tenant_id, permission_key, description)
SELECT v.id, t.id, v.permission_key, v.description
FROM tenants t
CROSS JOIN (VALUES
    ('40000000-0000-4000-8000-000000000001'::uuid, 'fleet:read'::citext, 'View fleet status and positions'),
    ('40000000-0000-4000-8000-000000000002'::uuid, 'vehicle:read'::citext, 'View vehicles'),
    ('40000000-0000-4000-8000-000000000003'::uuid, 'vehicle:write'::citext, 'Create and update vehicles'),
    ('40000000-0000-4000-8000-000000000004'::uuid, 'device:read'::citext, 'View tracking devices'),
    ('40000000-0000-4000-8000-000000000005'::uuid, 'device:write'::citext, 'Manage devices and assignments'),
    ('40000000-0000-4000-8000-000000000006'::uuid, 'geofence:read'::citext, 'View geofences'),
    ('40000000-0000-4000-8000-000000000007'::uuid, 'geofence:write'::citext, 'Create and update geofences'),
    ('40000000-0000-4000-8000-000000000008'::uuid, 'user:read'::citext, 'View tenant users'),
    ('40000000-0000-4000-8000-000000000009'::uuid, 'user:write'::citext, 'Create and update tenant users'),
    ('40000000-0000-4000-8000-000000000010'::uuid, 'role:manage'::citext, 'Manage roles and grants'),
    ('40000000-0000-4000-8000-000000000011'::uuid, 'audit:read'::citext, 'View tenant audit history')
) AS v(id, permission_key, description)
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, permission_key) DO UPDATE
SET description = EXCLUDED.description;

INSERT INTO user_roles (tenant_id, user_id, role_id, assigned_by)
SELECT t.id, u.id, r.id, u.id
FROM tenants t
JOIN users u ON u.tenant_id = t.id AND u.email = 'admin@fleet.local'
JOIN roles r ON r.tenant_id = t.id AND r.name = 'fleet_admin'
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING;

INSERT INTO role_permissions (tenant_id, role_id, permission_id)
SELECT t.id, r.id, p.id
FROM tenants t
JOIN roles r ON r.tenant_id = t.id
JOIN permissions p ON p.tenant_id = t.id
WHERE t.slug = 'nusantara-logistics'
  AND (
      r.name = 'fleet_admin'
      OR (r.name = 'dispatcher' AND p.permission_key IN (
          'fleet:read', 'vehicle:read', 'vehicle:write', 'device:read', 'device:write', 'geofence:read', 'geofence:write'
      ))
      OR (r.name = 'viewer' AND p.permission_key IN (
          'fleet:read', 'vehicle:read', 'device:read', 'geofence:read'
      ))
  )
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

INSERT INTO vehicles (
    id, tenant_id, license_plate, fleet_code, display_name, vehicle_type, status,
    make, model, manufacture_year, capacity_kg
)
SELECT v.id, t.id, v.license_plate, v.fleet_code, v.display_name, v.vehicle_type,
       v.status, v.make, v.model, v.manufacture_year, v.capacity_kg
FROM tenants t
CROSS JOIN (VALUES
    ('50000000-0000-4000-8000-000000000001'::uuid, 'B 9123 UXR'::citext, 'NL-001'::citext, 'Garuda 01', 'light_truck', 'in_transit', 'Mitsubishi', 'Fuso Canter', 2022::smallint, 3500.00::numeric),
    ('50000000-0000-4000-8000-000000000002'::uuid, 'B 8456 KLM'::citext, 'NL-002'::citext, 'Garuda 02', 'heavy_truck', 'available', 'Hino', 'Ranger 500', 2021::smallint, 10000.00::numeric),
    ('50000000-0000-4000-8000-000000000003'::uuid, 'B 7689 PQR'::citext, 'NL-003'::citext, 'Garuda 03', 'heavy_truck', 'in_transit', 'Isuzu', 'GIGA FVM', 2020::smallint, 15000.00::numeric),
    ('50000000-0000-4000-8000-000000000004'::uuid, 'B 1234 TUV'::citext, 'NL-004'::citext, 'Cendrawasih 04', 'van', 'available', 'Toyota', 'HiAce Premio', 2023::smallint, 1200.00::numeric),
    ('50000000-0000-4000-8000-000000000005'::uuid, 'B 5678 WXY'::citext, 'NL-005'::citext, 'Cendrawasih 05', 'pickup', 'in_transit', 'Suzuki', 'New Carry', 2022::smallint, 1000.00::numeric),
    ('50000000-0000-4000-8000-000000000006'::uuid, 'B 3344 NLA'::citext, 'NL-006'::citext, 'Komodo 06', 'pickup', 'maintenance', 'Mitsubishi', 'L300', 2021::smallint, 1200.00::numeric),
    ('50000000-0000-4000-8000-000000000007'::uuid, 'B 7788 JKT'::citext, 'NL-007'::citext, 'Komodo 07', 'light_truck', 'available', 'Isuzu', 'Traga', 2023::smallint, 2950.00::numeric),
    ('50000000-0000-4000-8000-000000000008'::uuid, 'B 9901 LOG'::citext, 'NL-008'::citext, 'Rinjani 08', 'light_truck', 'in_transit', 'Toyota', 'Dyna 136 HT', 2022::smallint, 4000.00::numeric)
) AS v(id, license_plate, fleet_code, display_name, vehicle_type, status, make, model, manufacture_year, capacity_kg)
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, license_plate) DO UPDATE
SET fleet_code = EXCLUDED.fleet_code,
    display_name = EXCLUDED.display_name,
    vehicle_type = EXCLUDED.vehicle_type,
    status = EXCLUDED.status,
    make = EXCLUDED.make,
    model = EXCLUDED.model,
    manufacture_year = EXCLUDED.manufacture_year,
    capacity_kg = EXCLUDED.capacity_kg,
    deleted_at = NULL;

INSERT INTO devices (id, tenant_id, serial_number, imei, label, status, last_seen_at, metadata)
SELECT d.id, t.id, d.serial_number, d.imei, d.label, d.status,
       CURRENT_TIMESTAMP - d.last_seen_ago, jsonb_build_object('firmware', d.firmware, 'protocol', 'teltonika')
FROM tenants t
CROSS JOIN (VALUES
    ('60000000-0000-4000-8000-000000000001'::uuid, 'FT-JKT-001'::citext, '867648041000001', 'Tracker NL-001', 'active', interval '1 minute', '03.29.00'),
    ('60000000-0000-4000-8000-000000000002'::uuid, 'FT-JKT-002'::citext, '867648041000002', 'Tracker NL-002', 'active', interval '4 minutes', '03.29.00'),
    ('60000000-0000-4000-8000-000000000003'::uuid, 'FT-JKT-003'::citext, '867648041000003', 'Tracker NL-003', 'active', interval '2 minutes', '03.29.00'),
    ('60000000-0000-4000-8000-000000000004'::uuid, 'FT-JKT-004'::citext, '867648041000004', 'Tracker NL-004', 'active', interval '7 minutes', '03.28.02'),
    ('60000000-0000-4000-8000-000000000005'::uuid, 'FT-JKT-005'::citext, '867648041000005', 'Tracker NL-005', 'active', interval '3 minutes', '03.29.00'),
    ('60000000-0000-4000-8000-000000000006'::uuid, 'FT-JKT-006'::citext, '867648041000006', 'Tracker NL-006', 'maintenance', interval '2 hours', '03.27.04'),
    ('60000000-0000-4000-8000-000000000007'::uuid, 'FT-JKT-007'::citext, '867648041000007', 'Tracker NL-007', 'active', interval '5 minutes', '03.29.00'),
    ('60000000-0000-4000-8000-000000000008'::uuid, 'FT-JKT-008'::citext, '867648041000008', 'Tracker NL-008', 'active', interval '1 minute', '03.29.00')
) AS d(id, serial_number, imei, label, status, last_seen_ago, firmware)
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, serial_number) DO UPDATE
SET imei = EXCLUDED.imei,
    label = EXCLUDED.label,
    status = EXCLUDED.status,
    last_seen_at = EXCLUDED.last_seen_at,
    metadata = EXCLUDED.metadata;

INSERT INTO device_assignments (id, tenant_id, device_id, vehicle_id, assigned_at)
SELECT m.id, t.id, d.id, v.id, CURRENT_TIMESTAMP - interval '30 days'
FROM tenants t
JOIN (VALUES
    ('70000000-0000-4000-8000-000000000001'::uuid, 'FT-JKT-001'::citext, 'NL-001'::citext),
    ('70000000-0000-4000-8000-000000000002'::uuid, 'FT-JKT-002'::citext, 'NL-002'::citext),
    ('70000000-0000-4000-8000-000000000003'::uuid, 'FT-JKT-003'::citext, 'NL-003'::citext),
    ('70000000-0000-4000-8000-000000000004'::uuid, 'FT-JKT-004'::citext, 'NL-004'::citext),
    ('70000000-0000-4000-8000-000000000005'::uuid, 'FT-JKT-005'::citext, 'NL-005'::citext),
    ('70000000-0000-4000-8000-000000000006'::uuid, 'FT-JKT-006'::citext, 'NL-006'::citext),
    ('70000000-0000-4000-8000-000000000007'::uuid, 'FT-JKT-007'::citext, 'NL-007'::citext),
    ('70000000-0000-4000-8000-000000000008'::uuid, 'FT-JKT-008'::citext, 'NL-008'::citext)
) AS m(id, serial_number, fleet_code) ON true
JOIN devices d ON d.tenant_id = t.id AND d.serial_number = m.serial_number
JOIN vehicles v ON v.tenant_id = t.id AND v.fleet_code = m.fleet_code
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, device_id) WHERE ended_at IS NULL DO UPDATE
SET vehicle_id = EXCLUDED.vehicle_id;

INSERT INTO vehicle_latest_positions (
    tenant_id, vehicle_id, device_id, position, speed_kph, heading_degrees,
    accuracy_m, ignition_on, recorded_at, received_at
)
SELECT t.id, v.id, d.id, ST_SetSRID(ST_MakePoint(p.longitude, p.latitude), 4326),
       p.speed_kph, p.heading_degrees, p.accuracy_m, p.ignition_on,
       CURRENT_TIMESTAMP - p.recorded_ago, CURRENT_TIMESTAMP
FROM tenants t
JOIN (VALUES
    ('NL-001'::citext, 'FT-JKT-001'::citext, 106.8456::double precision, -6.2088::double precision, 42.5::numeric, 86.0::numeric, 7.0::numeric, true, interval '1 minute'),
    ('NL-002'::citext, 'FT-JKT-002'::citext, 106.8272::double precision, -6.1754::double precision, 0.0::numeric, 12.0::numeric, 5.0::numeric, false, interval '4 minutes'),
    ('NL-003'::citext, 'FT-JKT-003'::citext, 106.8809::double precision, -6.1049::double precision, 31.2::numeric, 241.0::numeric, 8.0::numeric, true, interval '2 minutes'),
    ('NL-004'::citext, 'FT-JKT-004'::citext, 106.7816::double precision, -6.2444::double precision, 0.0::numeric, 180.0::numeric, 6.0::numeric, false, interval '7 minutes'),
    ('NL-005'::citext, 'FT-JKT-005'::citext, 106.9156::double precision, -6.2250::double precision, 27.8::numeric, 104.0::numeric, 9.0::numeric, true, interval '3 minutes'),
    ('NL-006'::citext, 'FT-JKT-006'::citext, 106.7990::double precision, -6.1900::double precision, 0.0::numeric, 0.0::numeric, 12.0::numeric, false, interval '2 hours'),
    ('NL-007'::citext, 'FT-JKT-007'::citext, 106.6559::double precision, -6.1256::double precision, 0.0::numeric, 320.0::numeric, 7.0::numeric, false, interval '5 minutes'),
    ('NL-008'::citext, 'FT-JKT-008'::citext, 106.9723::double precision, -6.2383::double precision, 55.1::numeric, 91.0::numeric, 6.0::numeric, true, interval '1 minute')
) AS p(fleet_code, serial_number, longitude, latitude, speed_kph, heading_degrees, accuracy_m, ignition_on, recorded_ago) ON true
JOIN vehicles v ON v.tenant_id = t.id AND v.fleet_code = p.fleet_code
JOIN devices d ON d.tenant_id = t.id AND d.serial_number = p.serial_number
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, vehicle_id) DO UPDATE
SET device_id = EXCLUDED.device_id,
    position = EXCLUDED.position,
    speed_kph = EXCLUDED.speed_kph,
    heading_degrees = EXCLUDED.heading_degrees,
    accuracy_m = EXCLUDED.accuracy_m,
    ignition_on = EXCLUDED.ignition_on,
    recorded_at = EXCLUDED.recorded_at,
    received_at = EXCLUDED.received_at;

INSERT INTO geofences (id, tenant_id, name, description, geofence_type, center, radius_m, is_active)
SELECT g.id, t.id, g.name, g.description, 'circle',
       ST_SetSRID(ST_MakePoint(g.longitude, g.latitude), 4326), g.radius_m, true
FROM tenants t
CROSS JOIN (VALUES
    ('80000000-0000-4000-8000-000000000001'::uuid, 'Central Jakarta Hub'::citext, 'Primary dispatch and consolidation hub', 106.8272::double precision, -6.1754::double precision, 5000.00::numeric),
    ('80000000-0000-4000-8000-000000000002'::uuid, 'Tanjung Priok Port'::citext, 'Port pickup and container staging zone', 106.8809::double precision, -6.1049::double precision, 3500.00::numeric),
    ('80000000-0000-4000-8000-000000000003'::uuid, 'Soekarno-Hatta Cargo'::citext, 'Airport cargo terminal operating zone', 106.6559::double precision, -6.1256::double precision, 4000.00::numeric)
) AS g(id, name, description, longitude, latitude, radius_m)
WHERE t.slug = 'nusantara-logistics'
ON CONFLICT (tenant_id, name) DO UPDATE
SET description = EXCLUDED.description,
    geofence_type = EXCLUDED.geofence_type,
    center = EXCLUDED.center,
    radius_m = EXCLUDED.radius_m,
    boundary = NULL,
    is_active = EXCLUDED.is_active;

COMMIT;
