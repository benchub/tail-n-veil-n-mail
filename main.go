package main

import (
  "fmt"
  "time"
  "os"
  "regexp"
  "strings"
  "strconv"
  "sync"
  "encoding/json"
  "flag"
)

import (
  "github.com/ActiveState/tail"
  _ "github.com/lib/pq"
  "database/sql"
)

type LogKey struct
{
  // what host the event came from
  host string
  
  // the syslog id, which gets recycled
  id uint64
}

type LogEvent struct
{
  // for concurrency protection
  sync.RWMutex
  
  // has the event completed
  closed bool
  
  // to identify this event uniquely for the events in flight map
  key LogKey
  
  // the event text so far (minus the syslog header)
  eventText string
  
  // the syslog time of the first line
  eventTimeStart time.Time
  
  // the syslog time of the last line
  eventTimeEnd time.Time
  
  // how many lines the event spanned
  lines uint64
  
  // A channel to help cull the event
  finish chan bool
  
  // if the first time we saw this event key was on a line >1, then
  // we know we don't have the full event. Record that with this
  fragment bool
  
  // Which bucket the event matched
  bucket string
}

// Every filter is handled by a distinct goroutine. This is the channel to send
// events to the first filter in the chain of filters.
var firstFilter = make(chan *LogEvent,100)

// When a filter goroutine matches, it sends the match to this channel
var eventsToReport = make(chan *LogEvent,1000)

// When we want to update the filter match count, we send the filter id to this channel
var filtersToMatch = make(chan int,1000)

// When we need to close off an event for processing, send the key here
var cull = make(chan LogKey,100)

// Some stats that we won't bother to make concurrency-safe.
// They're never decremented anyway.
var eventCount uint64
var lastEventAt time.Time
var warpTo tail.SeekInfo

// A dictionary of what events we're currently waiting on to timeout or evict for processing
var protectedEventsInFlight = struct{
  sync.RWMutex
  m map[LogKey]*LogEvent
} {m: make(map[LogKey]*LogEvent)}

// A function to take a new event and either merge it with one already in flight,
// or, if the eventKey isn't already in flight, make a new event.
func assembleRawLines(newlines chan *LogEvent) {
  // We'll need to be able to parse syslog [event-line] pairings
  re_logid, err := regexp.Compile(`[0-9]+-[0-9]+`)
  if err != nil {
    fmt.Fprintln(os.Stderr, "re_logid:", err)
  }

  for {
    line := <- newlines
    
    if line.eventText == "" {
      continue
    }
    
    // break out the stuff that syslog adds to every line
    syslogTokens := strings.SplitN(line.eventText," ",5)
    line.eventTimeStart, err = time.Parse(time.RFC3339Nano,syslogTokens[0])
    line.eventTimeEnd = line.eventTimeStart
    if err != nil {
      fmt.Fprintln(os.Stderr, "times are hard:", err)
      os.Exit(10)
    }
    line.key.host = syslogTokens[1]
    line.eventText = syslogTokens[4]
    
    logid := re_logid.FindString(syslogTokens[3])
    logidTokens := strings.Split(logid,"-")
    line.key.id, err = strconv.ParseUint(logidTokens[0],10,32)
    if err != nil {
      fmt.Fprintln(os.Stderr, "logids are hard:", err)
      os.Exit(11)
    }
    line.lines, err = strconv.ParseUint(logidTokens[1],10,32)
    if err != nil {
      fmt.Fprintln(os.Stderr, "logid lines are hard:", err)
      os.Exit(12)
    }
    
    protectedEventsInFlight.RLock()
    existingEvent, present := protectedEventsInFlight.m[line.key]
    protectedEventsInFlight.RUnlock()
    if present {
      if line.lines == 1 {
        // Looks like we're starting a new event for an existing event ID.
        // That means we need to evict and process the existing event with this key first.
        // However, we might be in the process of evicting that event *right now*,
        // so get an exclusive lock on it to make sure
        existingEvent.Lock()
        if !existingEvent.closed {
          // Looks like we haven't closed the finish channel, so tell cullStuff() to
          // kill this event off. That will be redundant if it's already happening, but
          // it won't harm anything if so.
          existingEvent.finish <- true
        }
        existingEvent.Unlock()
        
        // We need to block until it's actually gone, because we want to insert the same key
        // back for a different event
        _ = <- existingEvent.finish
        
        // Now start the new event
        beginEvent(line)
      } else {
        // Simply update the existing event.
        // No need for locking the protectedEventsInFlight map, as we already have the event we're going to munge
        // If we're in the process of culling the event..... well, that sucks. Guess we 
        // should have giving it more time.
        existingEvent.eventTimeEnd = line.eventTimeEnd
        existingEvent.eventText = existingEvent.eventText + "\n" + line.eventText
        existingEvent.lines++
      }
    } else {
      // New line in flight!
      if line.lines != 1 {
        // Looks like this event timed out too early.
        line.fragment = true
      }
      beginEvent(line)
    }
  }
}

var configFileFlag = flag.String("config", "", "the config file")
var logFileFlag = flag.String("log", "", "the log file")
var logFileOffsetFlag = flag.Int64("warp", 0, "open the log file to this offset")
var noIdleHandsFlag = flag.Bool("noIdleHands", false, "when set to true, kill us (ungracefully) if we seem to be doing nothing")

type Configuration struct {
  DBConn []string
  StatusInterval int
}

func main() {
  var db *sql.DB
  var status_interval int
  
  flag.Parse()
  
  if len(os.Args) == 1 {
    flag.PrintDefaults()
    os.Exit(0)
  }

  if *configFileFlag == "" {
    fmt.Println("I need a config file!")
    os.Exit(1)
  } else {
    configFile, err := os.Open(*configFileFlag)
    if err != nil {
      fmt.Println("opening config file:", err)
      os.Exit(2)
    }
    
    decoder := json.NewDecoder(configFile)
    configuration := &Configuration{}
    decoder.Decode(&configuration)
    
    db, err = sql.Open("postgres", configuration.DBConn[0])
    if err != nil {
      fmt.Println("couldn't connect to db", err)
      os.Exit(2)
    }
    
    status_interval = configuration.StatusInterval
  }

  if *logFileFlag == "" {
    fmt.Println("I need a log file!")
    os.Exit(1)
  }

  if *logFileOffsetFlag > 0 {
    warpTo.Offset = *logFileOffsetFlag
  } 

  // tail -F, all in go.
  // We poll instead of using inotify because cleanup is simplier 
  // and we'll always have more data anyway.
  logfile, err:= tail.TailFile(*logFileFlag, tail.Config {
    Location: &warpTo,
    Follow: true,
    Poll: true,
    ReOpen: true})
  if err != nil {
    fmt.Fprintln(os.Stderr, "couldn't tail file", err)
  }
  
  // start a goroutine to remove things that are ready to process from the inflight list
  go cullStuff(cull)
  
  // set up our filters goroutines
  setUpParsers(db)
  go updateFilterUsages(db)
  
  // when we play catch-up, we need to know when the most recent completed event was
  var mostRecentCompletedEvent time.Time
  err = db.QueryRow(`select coalesce(max(finished),'1970-01-01') from events`).Scan(&mostRecentCompletedEvent)
  if err != nil {
    fmt.Println("couldn't find most recent event", err)
    os.Exit(3)
  }

  
  // we like stats
  go reportProgress(*noIdleHandsFlag, status_interval, logfile)
  
  // set up a single goroutine to write to the db
  go reportEvent(db)
  
  // Lines need to be assembled in order, so only one assembler worker to look at lines,
  // but we can at least do that on another "thread" from where we're reading input and 
  // looking for newlines. Maybe that'll make us Go Moar Faster.
  // (Now that we're using the tail package this might be a waste.)
  rawlines := make(chan *LogEvent,1000)
  go assembleRawLines(rawlines)
  
  caughtUp := false
  fmt.Println("Beginning to catchup scan to", mostRecentCompletedEvent)
  for {
    newEvent := LogEvent{}
    line := <- logfile.Lines
      newEvent.eventText = line.Text
      
      if !caughtUp {
        tokens := strings.SplitN(newEvent.eventText," ",2)
        when, err := time.Parse(time.RFC3339Nano,tokens[0])
        if err != nil {
          fmt.Fprintln(os.Stderr, "times are hard when catching up:", err)
          os.Exit(13)
        }

        lastEventAt = when

        if mostRecentCompletedEvent.Before(when) {
          fmt.Println("Catchup complete!")
          caughtUp = true
          rawlines <- &newEvent
        }
      } else {
        rawlines <- &newEvent
      }
  }

  // until we implement graceful exiting, we'll never get here
  fmt.Println("done reading!")
}

func beginEvent(event *LogEvent) {
    event.finish = make(chan bool)

    protectedEventsInFlight.Lock()
    protectedEventsInFlight.m[event.key] = event
    protectedEventsInFlight.Unlock()

    go setFuse(event)
}

func setFuse(event *LogEvent) {
    // Give an event two seconds to complete
    // Note that this is wall time, which will be much longer than two seconds
    // of logged time, at least during catch up. Once we catch up to now(), we won't be able
    // to go faster than wall time, so this is good enough.

  select {
    case <- event.finish:
      // This event is being purged because a new matching eventKey has starting,
      // implying this event is certainly done and ready to be processed.
      cull <- event.key
    case <- time.After(2000 * time.Millisecond):
      // We haven't seen any more lines for this event in a while, so assume it's done.
      cull <- event.key
      _ = <- event.finish
  }
}

func processEvent(event *LogEvent) {
  eventCount++
  lastEventAt = event.eventTimeEnd
  
  // now that the event is all wrapped up and packaged,
  // send it into the filter chain
  firstFilter <- event
}

func cullStuff(c chan LogKey) {
  for {
    killMe := <-c
    
    protectedEventsInFlight.Lock()
    processMe, present := protectedEventsInFlight.m[killMe]
    if present {
      delete(protectedEventsInFlight.m,killMe)
    } else {
      // hm, that's wierd
      fmt.Println("shouldn't be here",killMe.id)
    }
    protectedEventsInFlight.Unlock()

    // at this point it's out of the in-flight map.
    // If we were waiting for that (to insert a new event for this same key),
    // notify the listener
    processMe.Lock()
    processMe.finish <- true
    
    // ...and we're done with this event, so let's never block on it again.
    close(processMe.finish)
    processMe.closed = true
    processMe.Unlock()

    processEvent(processMe)
  }
}

func reportProgress(noIdleHands bool, interval int, logfile *tail.Tail) {
  almostDead := false
  lastProcessed := eventCount

  for {
    protectedEventsInFlight.RLock()
    flying := len(protectedEventsInFlight.m)
    protectedEventsInFlight.RUnlock()

    fmt.Println(time.Now(),":",flying,"in flight,",eventCount,"processed so far,",warpTo.Offset,"seek, currently at:",lastEventAt)
    if (noIdleHands && lastProcessed == eventCount ) {
      if almostDead {
        var m map[string]int
      
        m["stacktracetime"] = 1
      } else {
        almostDead = true
      }
    } else {
      almostDead = false
    }
    
    lastProcessed = eventCount
    warpTo.Offset, _ = logfile.Tell()
    time.Sleep(time.Duration(interval) * time.Second)
  }
}
