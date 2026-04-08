-- Remove tenant_id from all tables (ChatAPI is single-tenant per deployment).
-- Also drops the unused tenants table and removes unused endpoint/metadata columns
-- from notification_subscriptions.
--
-- Each table is rebuilt using the rename→create→copy→drop pattern because
-- SQLite does not support DROP COLUMN on older versions and tenant_id is part
-- of some PRIMARY KEY definitions.

PRAGMA foreign_keys = OFF;

-- Drop the tenants table entirely — never queried.
DROP TABLE IF EXISTS tenants;

-- rooms: drop tenant_id
ALTER TABLE rooms RENAME TO _rooms_old;
CREATE TABLE rooms (
  room_id    TEXT PRIMARY KEY,
  type       TEXT NOT NULL,
  unique_key TEXT NULL,
  name       TEXT NULL,
  last_seq   INTEGER DEFAULT 0,
  metadata   JSON NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO rooms (room_id, type, unique_key, name, last_seq, metadata, created_at)
  SELECT room_id, type, unique_key, name, last_seq, metadata, created_at FROM _rooms_old;
DROP TABLE _rooms_old;
CREATE UNIQUE INDEX idx_rooms_unique_key ON rooms(unique_key) WHERE unique_key IS NOT NULL;

-- room_members: drop tenant_id
ALTER TABLE room_members RENAME TO _room_members_old;
CREATE TABLE room_members (
  chatroom_id TEXT NOT NULL,
  user_id     TEXT NOT NULL,
  role        TEXT DEFAULT 'member',
  joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (chatroom_id, user_id)
);
INSERT INTO room_members (chatroom_id, user_id, role, joined_at)
  SELECT chatroom_id, user_id, role, joined_at FROM _room_members_old;
DROP TABLE _room_members_old;
CREATE INDEX idx_members_user ON room_members(user_id);
CREATE INDEX idx_members_room ON room_members(chatroom_id);

-- messages: drop tenant_id
ALTER TABLE messages RENAME TO _messages_old;
CREATE TABLE messages (
  message_id  TEXT PRIMARY KEY,
  chatroom_id TEXT NOT NULL,
  sender_id   TEXT NOT NULL,
  seq         INTEGER NOT NULL,
  content     TEXT NOT NULL,
  meta        JSON NULL,
  created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO messages (message_id, chatroom_id, sender_id, seq, content, meta, created_at)
  SELECT message_id, chatroom_id, sender_id, seq, content, meta, created_at FROM _messages_old;
DROP TABLE _messages_old;
CREATE INDEX idx_messages_room_seq ON messages(chatroom_id, seq);

-- delivery_state: drop tenant_id, change PRIMARY KEY to (user_id, chatroom_id)
ALTER TABLE delivery_state RENAME TO _delivery_state_old;
CREATE TABLE delivery_state (
  user_id     TEXT NOT NULL,
  chatroom_id TEXT NOT NULL,
  last_ack    INTEGER DEFAULT 0,
  updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, chatroom_id)
);
INSERT INTO delivery_state (user_id, chatroom_id, last_ack, updated_at)
  SELECT user_id, chatroom_id, last_ack, updated_at FROM _delivery_state_old;
DROP TABLE _delivery_state_old;

-- undelivered_messages: drop tenant_id
ALTER TABLE undelivered_messages RENAME TO _undelivered_messages_old;
CREATE TABLE undelivered_messages (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id         TEXT NOT NULL,
  chatroom_id     TEXT NOT NULL,
  message_id      TEXT NOT NULL,
  seq             INTEGER NOT NULL,
  attempts        INTEGER DEFAULT 0,
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_attempt_at DATETIME NULL
);
INSERT INTO undelivered_messages (id, user_id, chatroom_id, message_id, seq, attempts, created_at, last_attempt_at)
  SELECT id, user_id, chatroom_id, message_id, seq, attempts, created_at, last_attempt_at FROM _undelivered_messages_old;
DROP TABLE _undelivered_messages_old;
CREATE INDEX idx_undelivered_user_room_seq ON undelivered_messages(user_id, chatroom_id, seq);
CREATE INDEX idx_undelivered_attempts ON undelivered_messages(attempts, created_at);

-- notifications: drop tenant_id
ALTER TABLE notifications RENAME TO _notifications_old;
CREATE TABLE notifications (
  notification_id TEXT PRIMARY KEY,
  topic           TEXT NOT NULL,
  payload         JSON NOT NULL,
  targets         JSON NULL,
  status          TEXT DEFAULT 'pending',
  attempts        INTEGER DEFAULT 0,
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_attempt_at DATETIME NULL
);
INSERT INTO notifications (notification_id, topic, payload, targets, status, attempts, created_at, last_attempt_at)
  SELECT notification_id, topic, payload, targets, status, attempts, created_at, last_attempt_at FROM _notifications_old;
DROP TABLE _notifications_old;
CREATE INDEX idx_notifications_status ON notifications(status, created_at);

-- notification_subscriptions: drop tenant_id, endpoint, metadata (never written or read)
ALTER TABLE notification_subscriptions RENAME TO _notif_subs_old;
CREATE TABLE notification_subscriptions (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  subscriber_id TEXT NOT NULL,
  topic         TEXT NOT NULL,
  created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO notification_subscriptions (id, subscriber_id, topic, created_at)
  SELECT id, subscriber_id, topic, created_at FROM _notif_subs_old;
DROP TABLE _notif_subs_old;
CREATE INDEX idx_notif_subs_subscriber_topic ON notification_subscriptions(subscriber_id, topic);

PRAGMA foreign_keys = ON;
