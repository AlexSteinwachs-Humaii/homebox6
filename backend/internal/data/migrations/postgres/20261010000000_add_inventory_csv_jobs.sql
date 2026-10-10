-- +goose Up
-- PostgreSQL names the original inline CHECK exports_kind_check.
ALTER TABLE exports DROP CONSTRAINT exports_kind_check;
ALTER TABLE exports ADD CONSTRAINT exports_kind_check
    CHECK (kind IN ('export', 'import', 'inventory_csv'));
