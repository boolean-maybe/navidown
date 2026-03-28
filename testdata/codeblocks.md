# Code Block Test

## Go

```go
package main

import "fmt"

// Renderable defines any component that can draw itself to a screen buffer.
type Renderable interface {
	Draw(screen tcell.Screen, x, y, width, height int)
	GetRect() (int, int, int, int)
	SetRect(x, y, width, height int)
}

func main() {
	fmt.Println("hello world")
}
```

## Python

```python
from dataclasses import dataclass

@dataclass
class Config:
    host: str = "localhost"
    port: int = 8080
    debug: bool = False

def serve(config: Config) -> None:
    """Start the server with the given configuration."""
    print(f"Listening on {config.host}:{config.port}")
    if config.debug:
        print("Debug mode enabled")
```

## JavaScript

```javascript
async function fetchData(url) {
  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    return await response.json();
  } catch (err) {
    console.error("fetch failed:", err.message);
    return null;
  }
}
```

## Bash

```bash
#!/bin/bash
set -euo pipefail

for file in *.md; do
  if [[ -f "$file" ]]; then
    echo "Processing: $file"
    wc -l "$file"
  fi
done
```

## JSON

```json
{
  "name": "navidown",
  "version": "1.0.0",
  "dependencies": {
    "glamour": "^0.7.0",
    "chroma": "^2.0.0"
  },
  "config": {
    "theme": "dark",
    "wordWrap": 80
  }
}
```

## SQL

```sql
SELECT u.name, COUNT(o.id) AS order_count
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.created_at > '2024-01-01'
GROUP BY u.name
HAVING COUNT(o.id) > 5
ORDER BY order_count DESC;
```

## SQL Schema Creation

```sql
CREATE TABLE users (
    id          SERIAL PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE orders (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      VARCHAR(50) NOT NULL DEFAULT 'pending',
    total       NUMERIC(10, 2) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status  ON orders(status);
```

## No language specified

```
This is a plain code block
with no language identifier.
It should still get borders and padding.
```

## Short single-line block

```go
fmt.Println("one liner")
```

## YAML

```yaml
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    environment:
      - NODE_ENV=production
    volumes:
      - ./data:/var/www/html
```

## Rust

```rust
fn main() {
    let numbers: Vec<i32> = (1..=10).collect();
    let sum: i32 = numbers.iter()
        .filter(|&&x| x % 2 == 0)
        .sum();
    println!("Sum of evens: {sum}");
}
```

## Very long lines

```go
func VeryLongFunctionName(parameterOne string, parameterTwo int, parameterThree bool, parameterFour map[string]interface{}) (string, error) {
	return fmt.Sprintf("This is an extremely long line that should test how the code block handles content wider than the available terminal width"), nil
}
```

## Empty code block

```go
```

## Diff

```diff
--- a/file.go
+++ b/file.go
@@ -1,5 +1,5 @@
 func main() {
-    fmt.Println("old")
+    fmt.Println("new")
 }
```
