-- name: GetFleetSnapshot :many
SELECT
    v.id AS vehicle_id,
    v.license_plate,
    v.fleet_code,
    v.display_name,
    v.vehicle_type,
    v.status,
    d.id AS device_id,
    d.serial_number,
    d.status AS device_status,
    p.recorded_at,
    p.received_at,
    ST_X(p.position)::double precision AS longitude,
    ST_Y(p.position)::double precision AS latitude,
    p.speed_kph,
    p.heading_degrees,
    p.accuracy_m,
    p.ignition_on
FROM vehicles v
LEFT JOIN vehicle_latest_positions p
  ON p.tenant_id = v.tenant_id
 AND p.vehicle_id = v.id
LEFT JOIN devices d
  ON d.tenant_id = p.tenant_id
 AND d.id = p.device_id
WHERE v.tenant_id = sqlc.arg(tenant_id)
  AND v.deleted_at IS NULL
ORDER BY v.fleet_code;

-- name: ListActiveGeofences :many
SELECT
    id,
    name,
    description,
    geofence_type,
    ST_X(center)::double precision AS center_longitude,
    ST_Y(center)::double precision AS center_latitude,
    radius_m,
    ST_AsGeoJSON(boundary)::text AS boundary_geojson,
    created_at,
    updated_at
FROM geofences
WHERE tenant_id = sqlc.arg(tenant_id)
  AND is_active
ORDER BY name;
