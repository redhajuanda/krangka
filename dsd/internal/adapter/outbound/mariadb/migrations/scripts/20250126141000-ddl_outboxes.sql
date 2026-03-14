-- +migrate Up

DROP TABLE IF EXISTS outboxes;
CREATE TABLE outboxes (
  id varchar(26) NOT NULL PRIMARY KEY,
  topic varchar(50) NOT NULL,
  payload json NOT NULL,
  status varchar(20) NOT NULL, -- pending, success, failed
  retry_attempt int NOT NULL DEFAULT 0,
  error_message TEXT,
  `target` varchar(50) NOT NULL, 
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at int NOT NULL DEFAULT 0
);

CREATE INDEX idx_outboxes_1 ON outboxes (status, deleted_at, retry_attempt, id);


-- +migrate Down

DROP TABLE IF EXISTS outboxes;
