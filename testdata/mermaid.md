# Mermaid Diagram Examples

- [Flowchart](#flowchart)
- [Sequence Diagram](#sequence-diagram)
- [Class Diagram](#class-diagram)
- [State Diagram](#state-diagram)
- [Entity Relationship Diagram](#entity-relationship-diagram)
- [Gantt Chart](#gantt-chart)
- [Pie Chart](#pie-chart)
- [Git Graph](#git-graph)
- [Journey Map](#journey-map)
- [Mindmap](#mindmap)
- [Timeline](#timeline)
- [Quadrant Chart](#quadrant-chart)
- [Sankey Diagram](#sankey-diagram)
- [Block Diagram](#block-diagram)


## Flowchart

```mermaid
flowchart TD
    A[Start] --> B{Is it working?}
    B -->|Yes| C[Great!]
    B -->|No| D[Debug]
    D --> E[Check logs]
    D --> F[Add breakpoints]
    E --> B
    F --> B
    C --> G[Deploy]
```

## Large Flowchart

```mermaid
flowchart TD
    A[User Request] --> B{Authenticated?}
    B -->|No| C[Show Login]
    C --> D[Enter Credentials]
    D --> E{Valid?}
    E -->|No| F[Show Error]
    F --> D
    E -->|Yes| G[Create Session]
    G --> H[Load Profile]
    B -->|Yes| H
    H --> I{Role?}
    I -->|Admin| J[Load Admin Dashboard]
    I -->|User| K[Load User Dashboard]
    I -->|Guest| L[Load Guest View]
    J --> M[Fetch Analytics]
    M --> N[Fetch User List]
    N --> O[Fetch System Logs]
    O --> P[Render Admin Panel]
    K --> Q[Fetch User Data]
    Q --> R[Fetch Notifications]
    R --> S[Fetch Recommendations]
    S --> T[Render User Panel]
    L --> U[Fetch Public Content]
    U --> V[Render Guest Panel]
    P --> W{Action?}
    T --> W
    V --> W
    W -->|Navigate| X[Route to Page]
    X --> Y[Load Page Data]
    Y --> Z[Render Page]
    Z --> W
    W -->|Logout| AA[Destroy Session]
    AA --> A
    W -->|API Call| AB[Validate Request]
    AB --> AC{Rate Limited?}
    AC -->|Yes| AD[Return 429]
    AC -->|No| AE[Process Request]
    AE --> AF{Success?}
    AF -->|Yes| AG[Return Response]
    AF -->|No| AH[Log Error]
    AH --> AI[Return 500]
    AG --> W
    AD --> W
    AI --> W
```

## Sequence Diagram

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Auth
    participant DB

    Client->>API: POST /login
    API->>Auth: Validate credentials
    Auth->>DB: Query user
    DB-->>Auth: User record
    Auth-->>API: Token
    API-->>Client: 200 OK + JWT

    Client->>API: GET /data (with JWT)
    API->>Auth: Verify token
    Auth-->>API: Valid
    API->>DB: Fetch data
    DB-->>API: Results
    API-->>Client: 200 OK + data
```

## Large Sequence Diagram

```mermaid
sequenceDiagram
    participant Browser
    participant CDN
    participant Gateway
    participant Auth
    participant UserSvc
    participant OrderSvc
    participant PaymentSvc
    participant Inventory
    participant DB
    participant Cache
    participant Queue
    participant Notify

    Browser->>CDN: GET /app.js
    CDN-->>Browser: Static assets

    Browser->>Gateway: POST /auth/login
    Gateway->>Auth: Validate credentials
    Auth->>DB: SELECT user WHERE email=?
    DB-->>Auth: User record
    Auth->>Cache: SET session:token
    Cache-->>Auth: OK
    Auth-->>Gateway: JWT token
    Gateway-->>Browser: 200 + JWT

    Browser->>Gateway: GET /orders (JWT)
    Gateway->>Auth: Verify token
    Auth->>Cache: GET session:token
    Cache-->>Auth: Session data
    Auth-->>Gateway: Valid
    Gateway->>OrderSvc: ListOrders(userId)
    OrderSvc->>DB: SELECT orders WHERE user_id=?
    DB-->>OrderSvc: Order records
    OrderSvc->>Cache: GET order:details:*
    Cache-->>OrderSvc: Cached details
    OrderSvc-->>Gateway: Order list
    Gateway-->>Browser: 200 + orders

    Browser->>Gateway: POST /orders/new
    Gateway->>Auth: Verify token
    Auth-->>Gateway: Valid
    Gateway->>OrderSvc: CreateOrder(items)
    OrderSvc->>Inventory: CheckStock(items)
    Inventory->>DB: SELECT stock WHERE product_id IN (?)
    DB-->>Inventory: Stock levels
    Inventory-->>OrderSvc: Stock confirmed

    OrderSvc->>PaymentSvc: ChargeCard(amount)
    PaymentSvc->>PaymentSvc: Validate card
    PaymentSvc-->>OrderSvc: Payment authorized

    OrderSvc->>DB: INSERT order
    DB-->>OrderSvc: Order created
    OrderSvc->>Inventory: ReserveStock(items)
    Inventory->>DB: UPDATE stock SET reserved=reserved+?
    DB-->>Inventory: Updated
    Inventory-->>OrderSvc: Reserved

    OrderSvc->>Queue: Publish(OrderCreated)
    Queue-->>OrderSvc: Acknowledged
    OrderSvc->>Cache: INVALIDATE order:list:userId
    Cache-->>OrderSvc: OK
    OrderSvc-->>Gateway: Order confirmation
    Gateway-->>Browser: 201 + order

    Queue->>Notify: Consume(OrderCreated)
    Notify->>Notify: Render email template
    Notify-->>Browser: Email: Order confirmed
```

## Class Diagram

```mermaid
classDiagram
    class Animal {
        +String name
        +int age
        +makeSound() void
    }
    class Dog {
        +String breed
        +fetch() void
    }
    class Cat {
        +bool indoor
        +purr() void
    }
    class Shelter {
        +List~Animal~ animals
        +adopt(Animal) bool
        +intake(Animal) void
    }

    Animal <|-- Dog
    Animal <|-- Cat
    Shelter "1" --> "*" Animal : houses
```

## Large Class Diagram

```mermaid
classDiagram
    class Application {
        -Config config
        -Router router
        -Logger logger
        +start() void
        +stop() void
        +healthCheck() Status
    }
    class Router {
        -List~Route~ routes
        -List~Middleware~ middleware
        +addRoute(Route) void
        +addMiddleware(Middleware) void
        +dispatch(Request) Response
    }
    class Middleware {
        <<interface>>
        +handle(Request, Next) Response
    }
    class AuthMiddleware {
        -TokenValidator validator
        +handle(Request, Next) Response
    }
    class RateLimiter {
        -int maxRequests
        -Duration window
        -Map~String,int~ counters
        +handle(Request, Next) Response
    }
    class Controller {
        <<abstract>>
        #Logger logger
        +handleRequest(Request) Response
    }
    class UserController {
        -UserService userService
        +getUser(id) Response
        +createUser(data) Response
        +updateUser(id, data) Response
        +deleteUser(id) Response
        +listUsers(filters) Response
    }
    class OrderController {
        -OrderService orderService
        -PaymentService paymentService
        +createOrder(data) Response
        +cancelOrder(id) Response
        +getOrderStatus(id) Response
    }
    class Service {
        <<interface>>
        +execute(Request) Result
    }
    class UserService {
        -UserRepository repo
        -EventBus events
        +findById(id) User
        +create(data) User
        +update(id, data) User
        +delete(id) void
    }
    class OrderService {
        -OrderRepository repo
        -InventoryService inventory
        -EventBus events
        +place(order) Order
        +cancel(id) void
        +status(id) OrderStatus
    }
    class Repository {
        <<interface>>
        +findById(id) Entity
        +save(entity) Entity
        +delete(id) void
        +findAll(query) List~Entity~
    }
    class UserRepository {
        -Database db
        +findById(id) User
        +findByEmail(email) User
        +save(user) User
        +delete(id) void
        +findAll(query) List~User~
    }
    class OrderRepository {
        -Database db
        +findById(id) Order
        +findByUser(userId) List~Order~
        +save(order) Order
        +delete(id) void
        +findAll(query) List~Order~
    }
    class Database {
        -ConnectionPool pool
        +query(sql, params) ResultSet
        +transaction(fn) Result
        +close() void
    }
    class EventBus {
        -Map~String,List~Handler~~ subscribers
        +publish(event) void
        +subscribe(topic, handler) void
        +unsubscribe(topic, handler) void
    }

    Application --> Router
    Application --> Logger
    Router --> Middleware
    Middleware <|.. AuthMiddleware
    Middleware <|.. RateLimiter
    Controller <|-- UserController
    Controller <|-- OrderController
    UserController --> UserService
    OrderController --> OrderService
    Service <|.. UserService
    Service <|.. OrderService
    UserService --> UserRepository
    UserService --> EventBus
    OrderService --> OrderRepository
    OrderService --> EventBus
    Repository <|.. UserRepository
    Repository <|.. OrderRepository
    UserRepository --> Database
    OrderRepository --> Database
```

## State Diagram

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Processing : Submit
    Processing --> Success : Valid
    Processing --> Error : Invalid
    Error --> Idle : Retry
    Success --> Idle : Reset
    Success --> [*]

    state Processing {
        [*] --> Validating
        Validating --> Transforming
        Transforming --> Saving
        Saving --> [*]
    }
```

## Large State Diagram

```mermaid
stateDiagram-v2
    [*] --> Idle

    Idle --> Connecting : Connect
    Connecting --> Authenticating : TCP established
    Connecting --> RetryWait : Connection failed
    RetryWait --> Connecting : Timer expired
    RetryWait --> Idle : Max retries

    Authenticating --> Handshake : Credentials valid
    Authenticating --> Idle : Auth rejected

    Handshake --> Ready : Handshake complete
    Handshake --> Idle : Handshake timeout

    Ready --> Subscribing : Subscribe
    Ready --> Publishing : Publish
    Ready --> Idle : Disconnect

    state Subscribing {
        [*] --> SendSub
        SendSub --> WaitSubAck
        WaitSubAck --> SubConfirmed : ACK received
        WaitSubAck --> SubFailed : Timeout
        SubFailed --> SendSub : Retry
        SubConfirmed --> [*]
    }

    Subscribing --> Ready : Complete

    state Publishing {
        [*] --> QueueMessage
        QueueMessage --> SendPublish
        SendPublish --> WaitPubAck
        WaitPubAck --> PubConfirmed : ACK received
        WaitPubAck --> PubRetry : Timeout
        PubRetry --> SendPublish : Retry
        PubRetry --> PubFailed : Max retries
        PubConfirmed --> [*]
        PubFailed --> [*]
    }

    Publishing --> Ready : Complete

    Ready --> Receiving : Message arrived

    state Receiving {
        [*] --> Validate
        Validate --> Deserialize : Valid
        Validate --> SendNack : Invalid
        Deserialize --> Dispatch
        Dispatch --> Process
        Process --> SendAck : Success
        Process --> SendNack : Error
        SendAck --> [*]
        SendNack --> [*]
    }

    Receiving --> Ready : Processed

    Ready --> Reconnecting : Connection lost

    state Reconnecting {
        [*] --> BackoffWait
        BackoffWait --> AttemptReconnect
        AttemptReconnect --> ReauthAttempt : TCP restored
        AttemptReconnect --> BackoffWait : Failed
        ReauthAttempt --> RestoreSubs : Auth OK
        ReauthAttempt --> BackoffWait : Auth failed
        RestoreSubs --> [*]
    }

    Reconnecting --> Ready : Restored
    Reconnecting --> Idle : Give up

    Ready --> Draining : Graceful shutdown
    Draining --> Closing : Queue empty
    Draining --> Closing : Drain timeout
    Closing --> [*]
```

## Entity Relationship Diagram

```mermaid
erDiagram
    CUSTOMER ||--o{ ORDER : places
    CUSTOMER {
        int id PK
        string name
        string email
    }
    ORDER ||--|{ LINE_ITEM : contains
    ORDER {
        int id PK
        date created
        string status
    }
    LINE_ITEM }|--|| PRODUCT : references
    LINE_ITEM {
        int quantity
        float price
    }
    PRODUCT {
        int id PK
        string name
        float unitPrice
    }
```

## Large ER Diagram

```mermaid
erDiagram
    TENANT ||--o{ USER : has
    TENANT {
        uuid id PK
        string name
        string plan
        date created_at
        bool active
    }
    USER ||--o{ USER_ROLE : has
    USER {
        uuid id PK
        uuid tenant_id FK
        string email
        string password_hash
        string first_name
        string last_name
        date last_login
        bool active
    }
    ROLE ||--o{ USER_ROLE : assigned
    ROLE ||--o{ ROLE_PERMISSION : grants
    ROLE {
        uuid id PK
        string name
        string description
    }
    USER_ROLE {
        uuid user_id FK
        uuid role_id FK
        date assigned_at
    }
    PERMISSION ||--o{ ROLE_PERMISSION : granted
    PERMISSION {
        uuid id PK
        string resource
        string action
        string description
    }
    ROLE_PERMISSION {
        uuid role_id FK
        uuid permission_id FK
    }
    USER ||--o{ PROJECT : owns
    USER ||--o{ TEAM_MEMBER : belongs
    PROJECT ||--o{ TASK : contains
    PROJECT ||--o{ SPRINT : has
    PROJECT {
        uuid id PK
        uuid owner_id FK
        uuid tenant_id FK
        string name
        string description
        string status
        date start_date
        date target_date
    }
    TEAM ||--o{ TEAM_MEMBER : includes
    TEAM ||--o{ PROJECT : assigned
    TEAM {
        uuid id PK
        uuid tenant_id FK
        string name
        string description
    }
    TEAM_MEMBER {
        uuid team_id FK
        uuid user_id FK
        string role
        date joined_at
    }
    SPRINT ||--o{ TASK : schedules
    SPRINT {
        uuid id PK
        uuid project_id FK
        string name
        int number
        date start_date
        date end_date
        string status
    }
    TASK ||--o{ COMMENT : has
    TASK ||--o{ TASK_ATTACHMENT : has
    TASK ||--o{ TASK_HISTORY : tracks
    TASK {
        uuid id PK
        uuid project_id FK
        uuid sprint_id FK
        uuid assignee_id FK
        uuid reporter_id FK
        string title
        string description
        string status
        string priority
        int story_points
        date created_at
        date updated_at
        date due_date
    }
    USER ||--o{ COMMENT : writes
    COMMENT {
        uuid id PK
        uuid task_id FK
        uuid author_id FK
        string body
        date created_at
        date updated_at
    }
    TASK_ATTACHMENT {
        uuid id PK
        uuid task_id FK
        uuid uploader_id FK
        string filename
        string content_type
        int size_bytes
        string storage_key
        date uploaded_at
    }
    TASK_HISTORY {
        uuid id PK
        uuid task_id FK
        uuid changed_by FK
        string field
        string old_value
        string new_value
        date changed_at
    }
```

## Gantt Chart

```mermaid
gantt
    title Project Timeline
    dateFormat YYYY-MM-DD
    section Design
        Requirements    :done, des1, 2024-01-01, 2024-01-15
        Wireframes      :done, des2, after des1, 10d
        UI Mockups      :active, des3, after des2, 15d
    section Development
        Backend API     :dev1, after des2, 30d
        Frontend        :dev2, after des3, 25d
        Integration     :dev3, after dev1, 10d
    section Testing
        Unit Tests      :test1, after dev1, 15d
        E2E Tests       :test2, after dev3, 10d
        UAT             :test3, after test2, 7d
```

## Pie Chart

```mermaid
pie title Language Distribution
    "Go" : 45
    "TypeScript" : 25
    "Python" : 15
    "Rust" : 10
    "Other" : 5
```

## Git Graph

```mermaid
gitGraph
    commit id: "init"
    commit id: "add readme"
    branch feature/auth
    checkout feature/auth
    commit id: "add login"
    commit id: "add JWT"
    checkout main
    branch feature/api
    commit id: "add routes"
    checkout main
    merge feature/auth id: "merge auth"
    merge feature/api id: "merge api"
    commit id: "release v1.0"
```

## Journey Map

```mermaid
journey
    title User Onboarding
    section Sign Up
        Visit landing page: 5: User
        Click sign up: 4: User
        Fill in form: 3: User
        Verify email: 2: User
    section First Use
        Complete tutorial: 3: User, System
        Create first project: 4: User
        Invite teammate: 3: User
    section Retention
        Daily check-in: 4: User
        Upgrade plan: 2: User
```

## Mindmap

```mermaid
mindmap
    root((Software Design))
        Patterns
            Creational
                Singleton
                Factory
                Builder
            Structural
                Adapter
                Decorator
            Behavioral
                Observer
                Strategy
        Principles
            SOLID
            DRY
            KISS
        Architecture
            Monolith
            Microservices
            Serverless
```

## Timeline

```mermaid
timeline
    title History of Web Frameworks
    2005 : Ruby on Rails
         : Django
    2009 : Node.js
         : AngularJS
    2013 : React
    2014 : Vue.js
    2016 : Angular 2
    2020 : Svelte 3
         : Next.js 10
    2023 : htmx revival
         : Server Components
```

## Quadrant Chart

```mermaid
quadrantChart
    title Tech Radar
    x-axis Low Effort --> High Effort
    y-axis Low Impact --> High Impact
    quadrant-1 Do First
    quadrant-2 Plan
    quadrant-3 Delegate
    quadrant-4 Eliminate
    Caching: [0.2, 0.8]
    Rewrite Auth: [0.8, 0.9]
    Fix Typos: [0.1, 0.1]
    Add Metrics: [0.3, 0.7]
    Migrate DB: [0.9, 0.6]
    Update Docs: [0.2, 0.3]
```

## Sankey Diagram

```mermaid
sankey-beta
    Source A,Process X,40
    Source A,Process Y,20
    Source B,Process X,30
    Source B,Process Z,10
    Process X,Output 1,50
    Process X,Output 2,20
    Process Y,Output 1,15
    Process Y,Output 3,5
    Process Z,Output 3,10
```

## Block Diagram

```mermaid
block-beta
    columns 3
    Frontend blockArrowId<["  "]>(right) Backend
    Backend blockArrowId2<["  "]>(right) Database

    space:3

    A["Load Balancer"]:3
```
