-- +goose Up
create table exports_inventory_csv
(
    id            uuid                       not null
        primary key,
    created_at    datetime                   not null,
    updated_at    datetime                   not null,
    kind          text     default 'export'  not null
        check (kind in ('export', 'import', 'inventory_csv')),
    status        text     default 'pending' not null
        check (status in ('pending', 'running', 'completed', 'failed')),
    progress      integer  default 0         not null,
    artifact_path text,
    size_bytes    integer  default 0         not null,
    error         text
        check (error is null or length(error) <= 1000),
    group_id      uuid                       not null
        constraint exports_groups_exports
            references groups
            on delete cascade
);

INSERT INTO exports_inventory_csv (id, created_at, updated_at, kind, status, progress, artifact_path, size_bytes, error, group_id)
SELECT id, created_at, updated_at, kind, status, progress, artifact_path, size_bytes, error, group_id FROM exports;
DROP TABLE exports;
ALTER TABLE exports_inventory_csv RENAME TO exports;

create index if not exists export_group_id
    on exports (group_id);

create index if not exists export_group_id_status
    on exports (group_id, status);
