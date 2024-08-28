# Notes

## Profiler

To enable pprof add HTTP server to the main function (for example):
```go
package main

import (
	"fmt"
	"log"
	_ "net/http/pprof".
)
func main() {
    // Code before.

    go func() {
        if err := http.ListenAndServe(":8090", nil); err != nil {
            log.Fatal(fmt.Errorf("http.ListenAndServe: %w", err))
        }
    }()

    // Code after.
}
```

### CPU profile

```bash
go tool pprof -http=":9090" -seconds=30 http://localhost:8090/debug/pprof/profile

# OR

curl -Ssv http://localhost:8090/debug/pprof/profile > profile.out
go tool pprof -http=":9090" -seconds=30 profile.out
```

### Memory profile

```bash
go tool pprof -http=":9090" -seconds=30 http://localhost:8090/debug/pprof/heap

# OR

curl -Ssv http://localhost:8090/debug/pprof/heap > heap.out
go tool pprof -http=":9090" -seconds=30 heap.out
```
