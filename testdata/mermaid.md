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
