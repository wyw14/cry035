CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS buildings (
    id text PRIMARY KEY,
    name text NOT NULL,
    address text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS equipment_models (
    id text PRIMARY KEY,
    manufacturer text NOT NULL,
    name text NOT NULL,
    category text NOT NULL CHECK (category IN ('elevator','freight_elevator','lifting_device'))
);

CREATE TABLE IF NOT EXISTS equipment (
    id text PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    building_id text NOT NULL REFERENCES buildings(id),
    model_id text NOT NULL REFERENCES equipment_models(id),
    location text NOT NULL,
    responsible_unit text NOT NULL,
    operating_status text NOT NULL CHECK (operating_status IN ('running','maintenance','limited','stopped')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS maintenance_programs (
    id text PRIMARY KEY,
    name text NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    statutory_cycle_days integer NOT NULL CHECK (statutory_cycle_days > 0),
    applicable_categories text[] NOT NULL,
    checklist jsonb NOT NULL CHECK (jsonb_typeof(checklist) = 'array'),
    active boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS maintenance_plans (
    id text PRIMARY KEY,
    equipment_id text NOT NULL REFERENCES equipment(id),
    program_id text NOT NULL REFERENCES maintenance_programs(id),
    program_version bigint NOT NULL,
    generation_key text,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    assignee text NOT NULL,
    spares jsonb NOT NULL DEFAULT '[]'::jsonb,
    status text NOT NULL CHECK (status IN ('planned','in_progress','suspended','overdue','pending_review','qualified','restricted','restored','cancelled')),
    ever_restricted boolean NOT NULL DEFAULT false,
    restriction_key text,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (window_end > window_start)
);

CREATE UNIQUE INDEX IF NOT EXISTS maintenance_plans_generation_key_uidx
    ON maintenance_plans(generation_key) WHERE generation_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS maintenance_plans_due_idx ON maintenance_plans(status, window_end);
CREATE INDEX IF NOT EXISTS maintenance_plans_equipment_idx ON maintenance_plans(equipment_id, window_start);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'maintenance_plans_no_overlap') THEN
        ALTER TABLE maintenance_plans ADD CONSTRAINT maintenance_plans_no_overlap
            EXCLUDE USING gist (
                equipment_id WITH =,
                tstzrange(window_start, window_end, '[)') WITH &&
            ) WHERE (status <> 'cancelled');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS executions (
    id text PRIMARY KEY,
    plan_id text NOT NULL UNIQUE REFERENCES maintenance_plans(id),
    technician text NOT NULL,
    checklist_snapshot jsonb NOT NULL,
    measurements jsonb NOT NULL,
    evidence jsonb NOT NULL,
    submitted_at timestamptz NOT NULL,
    review_comment text,
    review_outcome text CHECK (review_outcome IS NULL OR review_outcome IN ('passed','failed')),
    reviewer text,
    reviewed_at timestamptz
);

CREATE TABLE IF NOT EXISTS defects (
    id text PRIMARY KEY,
    equipment_id text NOT NULL REFERENCES equipment(id),
    plan_id text NOT NULL REFERENCES maintenance_plans(id),
    execution_id text NOT NULL REFERENCES executions(id),
    checklist_item_id text NOT NULL,
    level text NOT NULL CHECK (level IN ('minor','major','critical')),
    description text NOT NULL DEFAULT '',
    status text NOT NULL CHECK (status IN ('open','rectifying','ready_for_reinspection','closed')),
    due_at timestamptz NOT NULL,
    rectified_at timestamptz,
    closed_at timestamptz,
    version bigint NOT NULL DEFAULT 1,
    restriction_key text NOT NULL
);
CREATE INDEX IF NOT EXISTS defects_open_idx ON defects(equipment_id, status, due_at);

CREATE TABLE IF NOT EXISTS rectifications (
    id text PRIMARY KEY,
    defect_id text NOT NULL REFERENCES defects(id),
    action text NOT NULL,
    operator text NOT NULL,
    evidence_id text,
    created_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS reinspections (
    id text PRIMARY KEY,
    plan_id text NOT NULL REFERENCES maintenance_plans(id),
    equipment_id text NOT NULL REFERENCES equipment(id),
    restriction_key text NOT NULL,
    defect_ids text[] NOT NULL,
    inspector text NOT NULL,
    passed boolean NOT NULL,
    comment text NOT NULL DEFAULT '',
    inspected_at timestamptz NOT NULL,
    reviewer text,
    reviewed_at timestamptz
);

CREATE TABLE IF NOT EXISTS vendor_services (
    id text PRIMARY KEY,
    vendor_name text NOT NULL,
    equipment_id text NOT NULL REFERENCES equipment(id),
    plan_id text REFERENCES maintenance_plans(id),
    description text NOT NULL,
    amount_cents bigint NOT NULL CHECK (amount_cents >= 0),
    currency char(3) NOT NULL DEFAULT 'CNY',
    serviced_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_events (
    id text PRIMARY KEY,
    entity_type text NOT NULL,
    entity_id text NOT NULL,
    action text NOT NULL,
    actor text NOT NULL,
    request_id text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS audit_events_entity_idx ON audit_events(entity_id, occurred_at, id);

CREATE TABLE IF NOT EXISTS alerts (
    id text PRIMARY KEY,
    equipment_id text NOT NULL REFERENCES equipment(id),
    plan_id text REFERENCES maintenance_plans(id),
    level text NOT NULL CHECK (level IN ('info','warning','critical')),
    message text NOT NULL,
    is_read boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL
);

