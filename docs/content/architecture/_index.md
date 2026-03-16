---
title: "Architecture"
weight: 40
---

# Architecture Overview

ChatAPI is designed as a lightweight, production-ready chat service with a focus on simplicity, reliability, and multi-tenant operation.

## 🏛️ **System Architecture**

ChatAPI follows a service-oriented architecture with clear separation of concerns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Server   │    │  WebSocket      │    │ Background      │
│   (REST API)    │    │  Server         │    │   Workers       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Services      │
                    │   Layer         │
                    └─────────────────┘
                             │
                    ┌─────────────────┐
                    │   Database      │
                    │   (SQLite)      │
                    └─────────────────┘
```

## 📦 **Core Components**

### HTTP Server (REST API)
- **Framework**: Standard Go `net/http`
- **Endpoints**: RESTful API for chat operations
- **Middleware**: Authentication, rate limiting, CORS
- **Serialization**: JSON request/response handling

### WebSocket Server
- **Library**: Gorilla WebSocket
- **Protocol**: Custom JSON-based protocol
- **Features**: Real-time messaging, presence, typing indicators
- **Connection Management**: Per-user connection pooling

### Background Workers
- **Delivery Worker**: Processes undelivered messages and notifications
- **WAL Checkpoint Worker**: Manages SQLite WAL file growth
- **Scheduler**: Configurable intervals with graceful shutdown

### Services Layer
- **Tenant Service**: API key validation and tenant management
- **Chatroom Service**: Room creation and membership management
- **Message Service**: Message storage and retrieval
- **Realtime Service**: WebSocket connection and pub/sub
- **Notification Service**: Notification creation and delivery
- **Delivery Service**: Message delivery with retry logic

### Database Layer
- **Engine**: SQLite with WAL mode
- **Migrations**: Version-controlled schema updates
- **Connection Pooling**: Single-writer, multiple-reader pattern

## 🔐 **Security Model**

### Authentication
- **API Keys**: Tenant-level authentication via `X-API-Key` header only (never in query params — they appear in server logs)
- **API Key Storage**: SHA-256 hashes stored in the database (`api_key_hash` column); plaintext keys are returned exactly once at creation time and cannot be recovered
- **User Context**: User identification via `X-User-Id` header
- **WebSocket Auth**: Header-based (`X-API-Key` + `X-User-Id`) for server clients; token-based (`POST /ws/token` then `?token=`) for browser clients
- **No Sessions**: Stateless authentication for horizontal scaling

### Authorization
- **Room Membership**: Users can only access rooms they belong to
- **Message Ownership**: Users can only modify their own messages
- **Tenant Isolation**: Complete data isolation between tenants
- **Rate Limiting**: Per-tenant request throttling

### CORS

CORS headers for REST responses and WebSocket origin checks are both controlled by the `WS_ALLOWED_ORIGINS` environment variable. Set it to a comma-separated list of allowed origins (e.g. `https://app.example.com,https://admin.example.com`). Use `*` during local development only. When unset, browser-origin connections are rejected.

## 💾 **Database Schema**

### Core Tables

```sql
-- Tenants (multi-tenant isolation)
CREATE TABLE tenants (
  tenant_id TEXT PRIMARY KEY,
  api_key_hash TEXT UNIQUE NOT NULL,  -- SHA-256 of plaintext key; never stored in plain
  name TEXT,
  config JSON,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Rooms (chat rooms and channels)
CREATE TABLE rooms (
  room_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  type TEXT NOT NULL,           -- 'dm'|'group'|'channel'
  unique_key TEXT NULL,         -- deterministic key for DMs
  name TEXT NULL,
  metadata JSON NULL,           -- arbitrary app-level context (listing_id, order_id, etc.)
  last_seq INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Room membership
CREATE TABLE room_members (
  chatroom_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  role TEXT DEFAULT 'member',
  joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY(chatroom_id, user_id)
);

-- Messages with sequencing
CREATE TABLE messages (
  message_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  chatroom_id TEXT NOT NULL,
  sender_id TEXT NOT NULL,
  seq INTEGER NOT NULL,          -- per-room sequence
  content TEXT NOT NULL,
  meta TEXT NULL,                -- arbitrary JSON string attached to the message
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Delivery tracking
CREATE TABLE delivery_state (
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  chatroom_id TEXT NOT NULL,
  last_ack INTEGER DEFAULT 0,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (tenant_id, user_id, chatroom_id)
);

-- Undelivered message queue
CREATE TABLE undelivered_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  chatroom_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  attempts INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_attempt_at DATETIME NULL
);

-- Notifications system
CREATE TABLE notifications (
  notification_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  topic TEXT NOT NULL,
  payload JSON NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  status TEXT DEFAULT 'pending',
  attempts INTEGER DEFAULT 0,
  last_attempt_at DATETIME NULL
);
```

### Indexes and Performance

```sql
-- Room queries
CREATE INDEX idx_rooms_tenant ON rooms(tenant_id, room_id);
CREATE UNIQUE INDEX idx_rooms_unique_key ON rooms(tenant_id, unique_key);

-- Membership queries
CREATE INDEX idx_members_tenant_user ON room_members(tenant_id, user_id);
CREATE INDEX idx_members_tenant_room ON room_members(tenant_id, chatroom_id);

-- Message queries with pagination
CREATE INDEX idx_messages_room_seq ON messages(tenant_id, chatroom_id, seq);

-- Delivery state
CREATE INDEX idx_delivery_user_room ON delivery_state(tenant_id, user_id, chatroom_id);

-- Undelivered queue processing
CREATE INDEX idx_undelivered_user_room_seq ON undelivered_messages(tenant_id, user_id, chatroom_id, seq);
CREATE INDEX idx_undelivered_attempts ON undelivered_messages(tenant_id, attempts, created_at);

-- Notification processing
CREATE INDEX idx_notifications_status ON notifications(tenant_id, status, created_at);
```

## 🔄 **Data Flow**

### Message Sending Flow

1. **Client Request**: User sends message via REST or WebSocket
2. **Authentication**: Validate API key and user identity
3. **Rate Limiting**: Check tenant request limits
4. **Transaction**: Atomically increment room sequence and store message
5. **Real-time Delivery**: Broadcast to online room members
6. **Queue Undelivered**: Store for offline users
7. **Response**: Return message ID and sequence number

### Message Delivery Flow

1. **Store-then-Send**: Messages stored before delivery attempts
2. **Online Delivery**: Immediate WebSocket delivery for connected users
3. **Offline Queueing**: Persistent queue for disconnected users
4. **Retry Logic**: Exponential backoff with configurable limits
5. **Dead Letter**: Failed deliveries moved to dead-letter queue
6. **Cleanup**: Periodic removal of acknowledged messages

### WebSocket Connection Flow

1. **Connection**: Client establishes WebSocket connection
2. **Authentication**: Validate credentials (header-based or token-based)
3. **Registration**: Add to user's connection pool
4. **Presence Update**: Broadcast online status to room members
5. **Message Sync**: Stream missed messages since last ACK
6. **Real-time Events**: Bidirectional message exchange
7. **Disconnection**: Clean up with grace period for reconnection

## 🗂️ **Room Metadata**

Every room can carry an arbitrary JSON string in the `metadata` field, set at creation time via `POST /rooms`. This field is intended for app-level context that your application needs alongside a chat room — for example a marketplace listing ID, an order ID, a support ticket number, or any other domain identifier.

The field is stored as-is (a JSON string, not a parsed object) and returned on every room response. It is also forwarded in offline webhook payloads so your webhook handler can route or enrich events without an additional database lookup.

Example — creating a room with metadata:

```json
{
  "type": "dm",
  "members": ["buyer_42", "seller_7"],
  "metadata": "{\"listing_id\":\"lst_99\",\"order_id\":\"ord_42\"}"
}
```

## 📬 **Offline Webhooks**

When a message arrives for a user who has no active WebSocket connection, ChatAPI can POST a webhook to a URL configured on the tenant's `config.webhook_url` field. This lets your backend trigger push notifications, emails, or any other offline delivery channel without polling.

**Webhook payload:**

```json
{
  "event": "message.new",
  "tenant_id": "tenant_abc123",
  "room_id": "room_abc123",
  "recipient_id": "user2",
  "room_metadata": "{\"listing_id\":\"lst_99\",\"order_id\":\"ord_42\"}",
  "message": {
    "message_id": "msg_def456",
    "sender_id": "user1",
    "content": "Hello!",
    "seq": 44,
    "created_at": "2025-12-13T12:10:00Z"
  }
}
```

**Fields:**
- `event` — always `"message.new"` for offline message webhooks
- `tenant_id` — identifies the tenant
- `room_id` — the room where the message was sent
- `recipient_id` — the offline user who should receive the message
- `room_metadata` — the room's `metadata` string (may be `null`)
- `message` — the message that was sent

The webhook is delivered with a short timeout and retried with exponential backoff up to `RETRY_MAX_ATTEMPTS` times. Failed webhook deliveries appear in the dead-letter queue (`GET /admin/dead-letters`).

Configure `webhook_url` per tenant in the tenant's `config` JSON:

```json
{ "webhook_url": "https://your-app.example.com/chatapi-webhook" }
```

## 📊 **Performance Characteristics**

### Throughput
- **Messages/second**: 10,000+ (depending on hardware)
- **Concurrent Connections**: 100,000+ WebSocket connections
- **Database**: SQLite handles 1,000+ concurrent readers

### Latency
- **REST API**: <10ms typical response time
- **WebSocket Delivery**: <1ms for local delivery
- **Database Queries**: <5ms for typical operations

### Scalability Limits
- **Single Instance**: 10,000 concurrent users
- **Database Size**: 100GB+ SQLite databases supported
- **Memory Usage**: ~50MB base + 1KB per active WebSocket connection

## 🛡️ **Reliability Features**

### Data Durability
- **WAL Mode**: Write-ahead logging for crash recovery
- **Atomic Transactions**: Multi-table operations are atomic
- **Backup Strategy**: Hot backups with WAL preservation

### High Availability
- **Graceful Shutdown**: Clean connection draining
- **Connection Recovery**: Automatic reconnection with message sync
- **Health Checks**: Comprehensive service monitoring

### Error Handling
- **Retry Logic**: Configurable retry attempts with backoff
- **Circuit Breakers**: Prevent cascade failures
- **Dead Letter Queues**: Failed message handling

## 🔧 **Operational Aspects**

### Monitoring
- **Health Endpoint**: `GET /health` — service status and DB writability
- **Metrics Endpoint**: `GET /metrics` — live counters: `active_connections`, `messages_sent`, `dropped_broadcasts`, `delivery_attempts`, `delivery_failures`, `uptime_seconds`
- **Structured Logging**: JSON log output via `slog`
- **Admin Endpoints**: `GET /admin/dead-letters` for failed delivery inspection

### Configuration
- **Environment Variables**: Runtime configuration
- **Tenant Config**: Per-tenant feature flags (including `webhook_url`)
- **Database Tuning**: WAL and connection parameters

### Deployment
- **Single Binary**: Self-contained Go application
- **Docker Support**: Container-ready deployment
- **Systemd**: Service integration for Linux systems

## 🚀 **Extensibility**

### Plugin Architecture
- **Service Interfaces**: Clean interfaces for custom implementations
- **Middleware**: Extensible request processing pipeline
- **Event Hooks**: Integration points for custom logic

### API Evolution
- **Versioning**: URL-based API versioning
- **Backward Compatibility**: Non-breaking changes maintained
- **Deprecation Notices**: Clear migration paths

### Storage Backend
- **SQLite Default**: Embedded, zero-configuration
- **PostgreSQL**: Planned enterprise option
- **Custom Storage**: Interface-based abstraction

## 📈 **Scaling Strategy**

### Vertical Scaling
- **Memory**: Increase RAM for more connections
- **CPU**: More cores for concurrent processing
- **Storage**: Faster disks for database performance

### Horizontal Scaling (Future)
- **Load Balancer**: Distribute requests across instances
- **Shared Database**: PostgreSQL for multi-instance deployments
- **Pub/Sub**: NATS or Redis for cross-instance communication
- **Session Store**: Shared session state for WebSocket connections

### Multi-Region (Future)
- **Database Replication**: Cross-region data synchronization
- **CDN**: Static asset delivery
- **Global Load Balancing**: Geographic request routing

---

## ⚠️ **Known Limitations**

These are deliberate design trade-offs in v0.x. They are documented here so operators can plan accordingly.

### Single-Process WebSocket Registry

The in-memory WebSocket connection registry (`realtime.Service`) is local to a single process. Running multiple instances behind a load balancer **will not** deliver messages to users connected to a different instance — a user on instance A will not receive a broadcast sent via instance B.

**Impact**: ChatAPI is currently a single-instance deployment. This is by design for simplicity.

**Workaround**: Pin WebSocket connections to a single instance using sticky sessions (consistent hashing on `user_id` at the load balancer level).

**Future path**: Adding a pub/sub layer (Redis Pub/Sub or NATS) would allow fan-out across instances without changing the API surface.

### SQLite Concurrency Ceiling

SQLite with WAL mode supports high read concurrency but serialises all writes through a single writer connection. This is appropriate for most single-instance deployments but will become a bottleneck at very high write rates (sustained thousands of messages/second).

**Future path**: The service interface is designed so storage backends can be swapped. PostgreSQL support is planned.

### No TLS Termination

ChatAPI binds HTTP only. TLS must be handled by a reverse proxy in front of the service. See the [TLS guide](#tls--https) below.

---

## 🔒 **TLS / HTTPS**

ChatAPI does not handle TLS directly. Terminate TLS at a reverse proxy and forward to the service over localhost HTTP.

### Caddy (recommended — automatic HTTPS)

```
your.domain.com {
    reverse_proxy localhost:8080
}
```

### nginx

```nginx
server {
    listen 443 ssl;
    server_name your.domain.com;

    ssl_certificate     /etc/ssl/certs/your.crt;
    ssl_certificate_key /etc/ssl/private/your.key;

    location / {
        proxy_pass         http://localhost:8080;
        proxy_http_version 1.1;

        # Required for WebSocket upgrade
        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host       $host;
    }
}
```

### Docker with Caddy

```yaml
services:
  chatapi:
    image: hastenr/chatapi:latest
    environment:
      MASTER_API_KEY: "${MASTER_API_KEY}"
    volumes:
      - chatapi-data:/var/chatapi
    expose:
      - "8080"

  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data

volumes:
  chatapi-data:
  caddy-data:
```

---

*This architecture is designed for simplicity while maintaining production reliability. The service-oriented design allows for independent scaling and maintenance of components.*
