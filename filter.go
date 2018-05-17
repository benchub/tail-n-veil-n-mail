package main

import (
  "log"
  "regexp"
)

import (
  _ "github.com/lib/pq"
  "database/sql"
)

func setUpParsers(db *sql.DB) {
  // set up the first filter to ignore non-errors
  var lastFilter chan*LogEvent
  lastFilter = nil
  
  buckets, err := db.Query(`select id,name,workers,eat_it,report_it from buckets order by eat_it desc, workers desc`)
  if err != nil {
    log.Fatalln("couldn't select bucket list", err)
    // will now exit because Fatal
  }
  for buckets.Next() {
    var id,workers int
    var name string
    var eatIt,reportIt bool
    
    if err := buckets.Scan(&id,&name,&workers,&eatIt,&reportIt); err != nil {
      log.Fatalln("couldn't parse bucket row", err)
      // will now exit because Fatal
    }
    
    if lastFilter != nil {
      newFilter := setUpParsersForBucket(db, lastFilter,id,name,workers,eatIt,reportIt)
      lastFilter = newFilter
    } else {
      lastFilter = setUpParsersForBucket(db, firstFilter,id,name,workers,eatIt,reportIt)
    }
  }
  buckets.Close()

  go catchAll(lastFilter)
}

func setUpParsersForBucket(db *sql.DB, c chan *LogEvent, id int, name string, workers int, eatIt bool, reportIt bool) chan *LogEvent {
  lastFilter := c

  log.Println("Setting up filter for bucket",name)

  // get the list of hosts to restrict this bucket to, if we have one
  m := make(map[string]bool)
  onlyon, err := db.Query(`select host from onlyon where bucket_id=$1`, id)
  if err != nil {
    log.Fatalln("couldn't select onlyon hosts for", id, err)
    // will now exit because Fatal
  }
  for onlyon.Next() {
    var host string
    if err := onlyon.Scan(&host); err != nil {
      log.Fatalln("couldn't parse onlyon row for bucket", id, err)
      // will now exit because Fatal
    }
    
    m[host] = true
    
    log.Println("\tfor host",host)
  }
  if err := onlyon.Err(); err != nil {
    log.Fatalln("couldn't read onlyon hosts for", id, err)
    // will now exit because Fatal
  }
  onlyon.Close()

  // buckets can have multiple matching filters. Get them here.
  filters, err := db.Query(`select filter,report,id from filters where bucket_id=$1`, id)
  if err != nil {
    log.Fatalln("couldn't select filters for", id, err)
    // will now exit because Fatal
  }
  for filters.Next() {
    var filter string
    var fid int
    var report bool
    
    if err := filters.Scan(&filter, &report, &fid); err != nil {
      log.Fatalln("couldn't parse filter row for bucket", id, err)
      // will now exit because Fatal
    }
    
    
    log.Println("\tusing filter",filter)

    if len(m) == 0 {
      lastFilter = parseStuff(lastFilter,workers,name,filter,eatIt,reportIt,nil,fid,report)
    } else {
      lastFilter = parseStuff(lastFilter,workers,name,filter,eatIt,reportIt,m,fid,report)
    }
  }
  if err:= filters.Err(); err != nil {
    log.Fatalln("couldn't read filters for", id, err)
    // will now exit because Fatal
  }
  filters.Close()
  
  return lastFilter
}


func parseStuff(readFromHere chan *LogEvent, poolSize int, bucket string, match string, eatIt bool, reportIt bool, reportOnlyFor map[string]bool, filter_id int, updateCounts bool) chan *LogEvent {
  sendToHere := make(chan *LogEvent,100)
  
  for p := 0; p < poolSize; p++ {
    go func() {
      re, err := regexp.Compile(match)
      if err != nil {
        log.Println("regex compile error for", match, err)
      }
    
      for {
        event := <-readFromHere
      
        matched := re.MatchString(event.eventText)
        
        if matched && updateCounts {
          filtersToMatch <- filter_id
        }

        // we've matched, but will the bucket want it?
        reportForMe := true
        if reportOnlyFor != nil {
          _, present := reportOnlyFor[event.key.host]
          if !present {
            reportForMe = false
          }
        }

        if !matched || !reportForMe || (matched && !eatIt){
          // send it on
          sendToHere <- event
        } else {
          // process the match
          sendMatchToBucket(bucket,event,reportIt)
        }
      }
    }()
  }
  
  return sendToHere
}

func catchAll(readFromHere chan *LogEvent) {
  for {
    event := <- readFromHere
    sendMatchToBucket("",event,true)
  }
}

func updateFilterUsages(db *sql.DB) {
  for {
    id := <- filtersToMatch

    q, err := db.Query(`update filters set uses=uses+1 where id=$1`, id)
    if err != nil {
      log.Fatalln("couldn't update filter uses count for filter", id, err)
      // will now exit because Fatal
    }
    q.Close()
  }
}
