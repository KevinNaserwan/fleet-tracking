BEGIN;

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TABLE tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    slug citext NOT NULL,
    timezone text NOT NULL DEFAULT 'UTC',
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT tenants_slug_key UNIQUE (slug),
    CONSTRAINT tenants_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT tenants_slug_format CHECK (slug::text ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT tenants_timezone_not_blank CHECK (btrim(timezone) <> '')
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    email citext NOT NULL,
    password_hash text NOT NULL,
    full_name text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    last_login_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamptz,
    CONSTRAINT users_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT users_tenant_email_key UNIQUE (tenant_id, email),
    CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT users_email_not_blank CHECK (btrim(email::text) <> ''),
    CONSTRAINT users_password_is_argon2id CHECK (password_hash ~ '^\$argon2id\$v=[0-9]+\$'),
    CONSTRAINT users_full_name_not_blank CHECK (btrim(full_name) <> ''),
    CONSTRAINT users_status_valid CHECK (status IN ('active', 'disabled')),
    CONSTRAINT users_deleted_after_created CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

CREATE TABLE roles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    name citext NOT NULL,
    description text,
    is_system boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT roles_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT roles_tenant_name_key UNIQUE (tenant_id, name),
    CONSTRAINT roles_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT roles_name_format CHECK (name::text ~ '^[a-z][a-z0-9_]{1,62}$')
);

CREATE TABLE permissions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    permission_key citext NOT NULL,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT permissions_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT permissions_tenant_key_key UNIQUE (tenant_id, permission_key),
    CONSTRAINT permissions_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT permissions_key_format CHECK (permission_key::text ~ '^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*$'),
    CONSTRAINT permissions_description_not_blank CHECK (btrim(description) <> '')
);

CREATE TABLE user_roles (
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    assigned_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    assigned_by uuid,
    PRIMARY KEY (tenant_id, user_id, role_id),
    CONSTRAINT user_roles_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT user_roles_user_fk FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT user_roles_role_fk FOREIGN KEY (tenant_id, role_id) REFERENCES roles (tenant_id, id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT user_roles_assigned_by_fk FOREIGN KEY (tenant_id, assigned_by) REFERENCES users (tenant_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE role_permissions (
    tenant_id uuid NOT NULL,
    role_id uuid NOT NULL,
    permission_id uuid NOT NULL,
    granted_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, role_id, permission_id),
    CONSTRAINT role_permissions_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT role_permissions_role_fk FOREIGN KEY (tenant_id, role_id) REFERENCES roles (tenant_id, id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT role_permissions_permission_fk FOREIGN KEY (tenant_id, permission_id) REFERENCES permissions (tenant_id, id) ON UPDATE RESTRICT ON DELETE CASCADE
);

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    family_id uuid NOT NULL,
    token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by_ip inet,
    user_agent text,
    CONSTRAINT refresh_tokens_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT refresh_tokens_user_fk FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT refresh_tokens_hash_key UNIQUE (token_hash),
    CONSTRAINT refresh_tokens_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT refresh_tokens_hash_sha256_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT refresh_tokens_expiry_valid CHECK (expires_at > created_at),
    CONSTRAINT refresh_tokens_revocation_valid CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE TABLE vehicles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    license_plate citext NOT NULL,
    fleet_code citext NOT NULL,
    display_name text NOT NULL,
    vehicle_type text NOT NULL,
    status text NOT NULL DEFAULT 'available',
    make text,
    model text,
    manufacture_year smallint,
    vin citext,
    capacity_kg numeric(12, 2),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamptz,
    CONSTRAINT vehicles_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT vehicles_tenant_plate_key UNIQUE (tenant_id, license_plate),
    CONSTRAINT vehicles_tenant_fleet_code_key UNIQUE (tenant_id, fleet_code),
    CONSTRAINT vehicles_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT vehicles_plate_not_blank CHECK (btrim(license_plate::text) <> ''),
    CONSTRAINT vehicles_fleet_code_not_blank CHECK (btrim(fleet_code::text) <> ''),
    CONSTRAINT vehicles_display_name_not_blank CHECK (btrim(display_name) <> ''),
    CONSTRAINT vehicles_type_valid CHECK (vehicle_type IN ('heavy_truck', 'light_truck', 'van', 'pickup', 'car', 'motorcycle', 'other')),
    CONSTRAINT vehicles_status_valid CHECK (status IN ('available', 'in_transit', 'maintenance', 'inactive')),
    CONSTRAINT vehicles_year_valid CHECK (manufacture_year IS NULL OR manufacture_year BETWEEN 1900 AND 2100),
    CONSTRAINT vehicles_vin_valid CHECK (vin IS NULL OR vin::text ~ '^[A-HJ-NPR-Z0-9]{17}$'),
    CONSTRAINT vehicles_capacity_positive CHECK (capacity_kg IS NULL OR capacity_kg > 0),
    CONSTRAINT vehicles_deleted_after_created CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

CREATE UNIQUE INDEX vehicles_tenant_vin_key
    ON vehicles (tenant_id, vin)
    WHERE vin IS NOT NULL;

CREATE TABLE devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    serial_number citext NOT NULL,
    imei text,
    label text,
    status text NOT NULL DEFAULT 'active',
    last_seen_at timestamptz,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT devices_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT devices_tenant_serial_key UNIQUE (tenant_id, serial_number),
    CONSTRAINT devices_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT devices_serial_not_blank CHECK (btrim(serial_number::text) <> ''),
    CONSTRAINT devices_imei_format CHECK (imei IS NULL OR imei ~ '^[0-9]{14,16}$'),
    CONSTRAINT devices_status_valid CHECK (status IN ('active', 'offline', 'maintenance', 'retired')),
    CONSTRAINT devices_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX devices_tenant_imei_key
    ON devices (tenant_id, imei)
    WHERE imei IS NOT NULL;

CREATE TABLE device_assignments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    device_id uuid NOT NULL,
    vehicle_id uuid NOT NULL,
    assigned_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT device_assignments_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT device_assignments_device_fk FOREIGN KEY (tenant_id, device_id) REFERENCES devices (tenant_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT device_assignments_vehicle_fk FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT device_assignments_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT device_assignments_period_valid CHECK (ended_at IS NULL OR ended_at > assigned_at)
);

CREATE UNIQUE INDEX device_assignments_one_active_per_device
    ON device_assignments (tenant_id, device_id)
    WHERE ended_at IS NULL;

CREATE INDEX device_assignments_active_vehicle_idx
    ON device_assignments (tenant_id, vehicle_id)
    WHERE ended_at IS NULL;

CREATE INDEX device_assignments_history_idx
    ON device_assignments (tenant_id, device_id, assigned_at DESC);

CREATE TABLE vehicle_latest_positions (
    tenant_id uuid NOT NULL,
    vehicle_id uuid NOT NULL,
    device_id uuid NOT NULL,
    position geometry(Point, 4326) NOT NULL,
    speed_kph numeric(7, 2),
    heading_degrees numeric(6, 2),
    altitude_m numeric(10, 2),
    accuracy_m numeric(10, 2),
    ignition_on boolean,
    recorded_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, vehicle_id),
    CONSTRAINT vehicle_latest_positions_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT vehicle_latest_positions_vehicle_fk FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id) ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT vehicle_latest_positions_device_fk FOREIGN KEY (tenant_id, device_id) REFERENCES devices (tenant_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT vehicle_latest_positions_srid CHECK (ST_SRID(position) = 4326),
    CONSTRAINT vehicle_latest_positions_not_empty CHECK (NOT ST_IsEmpty(position)),
    CONSTRAINT vehicle_latest_positions_speed_valid CHECK (speed_kph IS NULL OR speed_kph BETWEEN 0 AND 350),
    CONSTRAINT vehicle_latest_positions_heading_valid CHECK (heading_degrees IS NULL OR heading_degrees >= 0 AND heading_degrees < 360),
    CONSTRAINT vehicle_latest_positions_accuracy_valid CHECK (accuracy_m IS NULL OR accuracy_m >= 0),
    CONSTRAINT vehicle_latest_positions_receive_order CHECK (received_at >= recorded_at)
);

CREATE INDEX vehicle_latest_positions_position_gix
    ON vehicle_latest_positions USING gist (position);

CREATE INDEX vehicle_latest_positions_recorded_idx
    ON vehicle_latest_positions (tenant_id, recorded_at DESC);

CREATE TABLE geofences (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    name citext NOT NULL,
    description text,
    geofence_type text NOT NULL,
    center geometry(Point, 4326),
    radius_m numeric(12, 2),
    boundary geometry(Geometry, 4326),
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT geofences_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT geofences_tenant_name_key UNIQUE (tenant_id, name),
    CONSTRAINT geofences_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT geofences_name_not_blank CHECK (btrim(name::text) <> ''),
    CONSTRAINT geofences_type_valid CHECK (geofence_type IN ('circle', 'polygon')),
    CONSTRAINT geofences_shape_valid CHECK (
        (geofence_type = 'circle' AND center IS NOT NULL AND radius_m > 0 AND boundary IS NULL)
        OR
        (geofence_type = 'polygon' AND center IS NULL AND radius_m IS NULL AND boundary IS NOT NULL
            AND ST_GeometryType(boundary) IN ('ST_Polygon', 'ST_MultiPolygon')
            AND ST_IsValid(boundary) AND NOT ST_IsEmpty(boundary))
    ),
    CONSTRAINT geofences_center_srid CHECK (center IS NULL OR ST_SRID(center) = 4326),
    CONSTRAINT geofences_boundary_srid CHECK (boundary IS NULL OR ST_SRID(boundary) = 4326)
);

CREATE INDEX geofences_center_gix
    ON geofences USING gist (center)
    WHERE center IS NOT NULL;

CREATE INDEX geofences_boundary_gix
    ON geofences USING gist (boundary)
    WHERE boundary IS NOT NULL;

CREATE INDEX geofences_active_idx
    ON geofences (tenant_id, name)
    WHERE is_active;

CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    actor_user_id uuid,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id uuid,
    request_id text,
    source_ip inet,
    user_agent text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT audit_logs_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT audit_logs_actor_fk FOREIGN KEY (tenant_id, actor_user_id) REFERENCES users (tenant_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT audit_logs_action_not_blank CHECK (btrim(action) <> ''),
    CONSTRAINT audit_logs_entity_type_not_blank CHECK (btrim(entity_type) <> ''),
    CONSTRAINT audit_logs_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX users_tenant_status_idx ON users (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX user_roles_role_idx ON user_roles (tenant_id, role_id, user_id);
CREATE INDEX role_permissions_permission_idx ON role_permissions (tenant_id, permission_id, role_id);
CREATE INDEX refresh_tokens_user_active_idx ON refresh_tokens (tenant_id, user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX refresh_tokens_family_active_idx ON refresh_tokens (tenant_id, family_id) WHERE revoked_at IS NULL;
CREATE INDEX vehicles_tenant_status_idx ON vehicles (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX devices_tenant_status_idx ON devices (tenant_id, status);
CREATE INDEX audit_logs_tenant_occurred_idx ON audit_logs (tenant_id, occurred_at DESC);
CREATE INDEX audit_logs_entity_idx ON audit_logs (tenant_id, entity_type, entity_id, occurred_at DESC);

CREATE TRIGGER tenants_set_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER roles_set_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER permissions_set_updated_at BEFORE UPDATE ON permissions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER vehicles_set_updated_at BEFORE UPDATE ON vehicles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER devices_set_updated_at BEFORE UPDATE ON devices
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER device_assignments_set_updated_at BEFORE UPDATE ON device_assignments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER geofences_set_updated_at BEFORE UPDATE ON geofences
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
