package main

import (
  "fmt"
  "os"
)

import (
  _ "github.com/lib/pq"
  "database/sql"
)

// a goroutine to record matching events
func reportEvent(db *sql.DB, worker string) {
  for {
    event := <- eventsToReport
    
    if event.bucket != "" {
      r, err := db.Query(`INSERT INTO events(bucket_id, host, event, started, finished, lines, fragment, worker) VALUES((select id from buckets where name = $1), $2, $3, $4, $5, $6, $7, $8)`, event.bucket, event.key.host, event.eventText, event.eventTimeStart, event.eventTimeEnd, event.lines, event.fragment, worker)
      if err != nil {
        fmt.Println("couldn't insert new event for bucket", event.bucket, "at", event.eventTimeStart, err)
        os.Exit(3)
      }
      r.Close()
    } else {
      r, err := db.Query(`INSERT INTO events(bucket_id, host, event, started, finished, lines, fragment, worker) VALUES(null, $1, $2, $3, $4, $5, $6, $7)`, event.key.host, event.eventText, event.eventTimeStart, event.eventTimeEnd, event.lines, event.fragment, worker)
      if err != nil {
        fmt.Println("couldn't insert new event for catchall bucket at", event.eventTimeStart, err)
        os.Exit(3)
      }
      r.Close()
    }
  }
}


// Just because a function matches a bucket, we might not want to actually record it;
// this function handles that logic.
func sendMatchToBucket(bucket string, event *LogEvent, reportIt bool) {
  if reportIt && !event.fragment {
    // fmt.Println("bucket",bucket,"gets event at",event.eventTimeEnd)
    event.bucket = bucket
    eventsToReport <- event
  }
}
