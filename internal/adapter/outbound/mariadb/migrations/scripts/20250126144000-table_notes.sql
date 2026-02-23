-- +migrate Up

CREATE TABLE IF NOT EXISTS notes (
  id varchar(26) PRIMARY KEY NOT NULL,
  title varchar(255) NOT NULL,
  content text NOT NULL,
  created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  deleted_at int NOT NULL DEFAULT 0
);

-- +migrate Down

DROP TABLE IF EXISTS notes;