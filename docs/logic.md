# アプリケーションロジック

```mermaid
graph TD
    subgraph Discord Interaction
        A[User sends message/command] --> B{Event Type?};
    end

    B -- "/chat" Slash Command --> C[chatCommandHandler];
    B -- DM or Reply to Bot --> D[messageCreateHandler];

    C --> E[chat.Service.GetResponse];
    D --> E;

    subgraph Core Logic in chat.Service
        E --> F{1. Bot Message?};
        F -- Yes --> G[Check Bot conversation limit (max 3) & Force Ollama];
        F -- No --> H[2. Build Prompt with History];
        G --> H;

        H --> I{3. Determine LLM};
        I -- Ollama Enabled --> J[getOllamaResponse];
        I -- Gemini --> K[Call Gemini API];

        K --> L{4a. Quota Exceeded?};
        L -- Yes --> M{Fallback?};
        M -- Secondary Gemini --> K;
        M -- Ollama --> J;
        M -- No --> N[Return Error];

        L -- No --> O{4b. Function Call?};
        O -- Yes --> P[Execute Function (Weather/URL)];
        P --> Q[Call Gemini API again with tool result];
        Q --> R[Final Response];
        O -- No --> R;
        J --> R;
    end

    subgraph Data Persistence & Response
        R --> S[Add conversation to History (DuckDB)];
        S --> T[Send Response to Discord];
    end

    style Core Logic in chat.Service fill:#f9f,stroke:#333,stroke-width:2px
    style Discord Interaction fill:#ccf,stroke:#333,stroke-width:2px
    style Data Persistence & Response fill:#cfc,stroke:#333,stroke-width:2px
