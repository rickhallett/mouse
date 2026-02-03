# Architecture

## ASCII Diagram

```
                         +-----------------+
                         |   Telegram API  |
                         +--------+--------+
                                  |
                                  v
+----------------+        +-------+--------+        +-------------------+
|  mousectl CLI  |------->|    Gateway    |------->|  Approvals API    |
+----------------+  HTTP  |  (HTTP mux)   |  HTTP  +-------------------+
                                  |
                                  |
                                  v
                           +------+-------+
                           | Orchestrator |
                           +------+-------+
                                  |
                +-----------------+------------------+
                |                 |                  |
                v                 v                  v
         +-------------+   +--------------+   +---------------+
         |   Sessions  |   |    SQLite    |   |     LLM       |
         |  (Markdown) |   |  (state/db)  |   |  (Anthropic)  |
         +------+------+   +------+-------+   +-------+-------+
                |                 |                   |
                |                 |                   v
                |                 |            +------+-------+
                |                 |            | Telegram Out |
                |                 |            +--------------+
                |                 |
                v                 v
         +-------------+   +--------------+
         |  Indexer    |<--|  Index Tables|
         +------+------+   +--------------+
                |
                v
         +-------------+
         |  Search API |
         +-------------+

         +-------------+
         |  Tools API  |-----> Docker Sandbox (network=none, RO root)
         +-------------+

         +-------------+
         |   Cron      |-----> Sessions + SQLite + LLM
         +-------------+

         +-------------+
         |   Logging   |-----> runtime/logs/mouse.log (JSONL)
         +-------------+
```

## Mermaid Diagram

```mermaid
flowchart TB
    TG[Telegram API] --> GW[Gateway HTTP mux]
    MCTL[mousectl CLI] -->|HTTP| GW
    GW --> AP[Approvals API]

    GW --> ORCH[Orchestrator]
    ORCH --> SESS[Sessions (Markdown)]
    ORCH --> DB[SQLite state DB]
    ORCH --> LLM[LLM Provider]
    LLM --> TGOUT[Telegram Outbound]

    subgraph Indexer
      IDX[Index Worker] --> IDXDB[Index Tables]
      IDXDB --> SEARCH[Search API]
    end
    DB --> IDXDB
    GW --> SEARCH
    GW --> REINDEX[Reindex API]
    REINDEX --> IDX

    subgraph Sandbox
      TOOLS[Tools API] --> DOCKER[Docker Sandbox
      RO root, network=none, RW workspace bind]
    end
    GW --> TOOLS

    subgraph Cron
      CRON[Cron Scheduler] --> SESS
      CRON --> DB
      CRON --> LLM
    end
    GW --> CRON

    GW --> LOGS[JSONL Logs -> runtime/logs/mouse.log]
```
