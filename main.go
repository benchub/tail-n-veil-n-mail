package main

import (
  "bytes"
  "errors"
  "log"
  "fmt"
  "time"
  "os"
  "syscall"
  "os/exec"
  "os/signal"
  "regexp"
  "strings"
  "strconv"
  "sync"
  "encoding/json"
  "flag"
  "runtime/pprof"
  "runtime"
)

import (
  "github.com/hpcloud/tail"
  _ "github.com/lib/pq"
  "database/sql"
)

type LogKey struct
{
  // what host the event came from
  host string
  
  // what pid the event came from
  pid uint64

  // the syslog id, which gets recycled
  id uint64
}

type PoorMansTime struct
{
  // a pointerless version of time.Time, in an attempt to reduce GC activity.
  // We will assume all times are in UTC (which is just God's Time anyway)
  sec int64
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
  eventTimeStart PoorMansTime
  
  // the syslog time of the last line
  eventTimeEnd PoorMansTime

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

// We want to normalize the query text we get, asynchronously. Send event text here.
var normalizeThese = make(chan bytes.Buffer,1000)

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
var lastEventAt PoorMansTime
var warpTo tail.SeekInfo

// A set of hosts that we don't want to process any events for
var ignoreTheseHosts = make(map[string]struct{})

// A dictionary of what events we're currently waiting on to timeout or evict for processing
var protectedEventsInFlight = struct{
  sync.RWMutex
  m map[LogKey]*LogEvent
} {m: make(map[LogKey]*LogEvent)}

var re_logid,_ = regexp.Compile(`[0-9]+-[0-9]+`)
var re_pidid,_ = regexp.Compile(`\[(\d+)\]`)
var re_leading_whitespace,_ = regexp.Compile(`^((#011)|(\s+))`)

type RawLine struct {
  t PoorMansTime
  host string
  text string
  pid uint64
  keyID uint64
  keyLine uint64
}


func decodeSyslogPrefix(blob string) (RawLine,error) {
  r := RawLine{}
  var err error
  var inefficientTime time.Time

  // break out the stuff that syslog adds to every line
  syslogTokens := strings.SplitN(blob," ",5)
  if len(syslogTokens) != 5 {
    return RawLine{}, errors.New("couldn't split line into 5 tokens")
  }
  inefficientTime, err = time.Parse(time.RFC3339Nano,syslogTokens[0])
  if err != nil {
    return RawLine{}, errors.New("couldn't parse time")
  }
  r.t.sec = inefficientTime.Unix()
  r.host = syslogTokens[1]
  r.text = syslogTokens[4]
    
  logid := re_logid.FindString(syslogTokens[3])
  logidTokens := strings.Split(logid,"-")
  if len(logidTokens) != 2 {
    return RawLine{}, errors.New("couldn't split logid into 2 tokens")
  }
  r.keyID, err = strconv.ParseUint(logidTokens[0],10,32)
  if err != nil {
    return RawLine{}, errors.New("Couldn't parse keyID")
  }
  if re_pidid.MatchString(syslogTokens[2]) {
    matches := re_pidid.FindStringSubmatch(syslogTokens[2])
    r.pid, err = strconv.ParseUint(matches[1],10,32)
    if err != nil {
      log.Println("pid",matches[1],"isn't numeric?")
      return RawLine{}, errors.New("Invalid pid")
    }
  } else {
    log.Println("no pid in",syslogTokens[2])
    return RawLine{}, errors.New("Couldn't parse pid")
  }
  r.keyLine, err = strconv.ParseUint(logidTokens[1],10,32)
  if err != nil {
    return RawLine{}, errors.New("Couldn't parse keyLine")
  }

  return r,nil
}

// A function to take a new event and either merge it with one already in flight,
// or, if the eventKey isn't already in flight, make a new event.
func assembleRawLines(newlines chan *LogEvent) {
  for {
    line := <- newlines
    
    if line.eventText == "" {
      continue
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
        startsWithWhitespace := re_leading_whitespace.MatchString(line.eventText)

        if startsWithWhitespace {
          existingEvent.eventText = existingEvent.eventText + "\n" + line.eventText
        } else {
          existingEvent.eventText = existingEvent.eventText + line.eventText
        }
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
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")

type Configuration struct {
  DBConn []string
  StatusInterval int
  EmailInterval int
  EmailsTo []string
  EmailSubject string
  EmailHeader string
  Worker string
}

func main() {
  var db *sql.DB
  var status_interval int
  var email_interval int
  var send_alerts_to []string
  var email_subject string
  var email_header string
  var worker string

  flag.Parse()
  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
  }

  if len(os.Args) == 1 {
    flag.PrintDefaults()
    os.Exit(0)
  }

  sigs := make(chan os.Signal, 1)
  // catch all signals since not explicitly listing
  signal.Notify(sigs,syscall.SIGQUIT,syscall.SIGTERM,syscall.SIGINT)
  //signal.Notify(sigs)
  // method invoked upon seeing signal
  go func() {
    s := <-sigs
    log.Printf("RECEIVED SIGNAL: %s",s)
    AppCleanup()
    os.Exit(1)
  }()

  if *configFileFlag == "" {
    log.Fatalln("I need a config file!")
    // will now exit because Fatal
  } else {
    configFile, err := os.Open(*configFileFlag)
    if err != nil {
      log.Fatalln("opening config file:", err)
      // will now exit because Fatal
    }
    
    decoder := json.NewDecoder(configFile)
    configuration := &Configuration{}
    decoder.Decode(&configuration)
    
    db, err = sql.Open("postgres", configuration.DBConn[0])
    if err != nil {
      log.Fatalln("couldn't connect to db", err)
      // will now exit because Fatal
    }
    
    status_interval = configuration.StatusInterval
    email_interval = configuration.EmailInterval
    send_alerts_to = configuration.EmailsTo
    email_subject = configuration.EmailSubject
    email_header = configuration.EmailHeader
    worker = configuration.Worker

    ignoreBlacklistedHosts(db)

    // if we see configuration changes, we need to know about it
    go watchForConfigChanges(db, configuration.DBConn[0], os.Args)
  }

  if *logFileFlag == "" {
    log.Fatalln("I need a log file!")
    // will now exit because Fatal
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
    log.Fatalln("couldn't tail file", err)
    // will now exit because Fatal
  }
  
  // start a goroutine to remove things that are ready to process from the inflight list
  go cullStuff(cull)
  
  // start a goroutine to normalize the events we see
  go normalizeEvents(db, normalizeThese)

  // set up our filter goroutines
  setUpParsers(db)
  go updateFilterUsages(db)
  
  // when we play catch-up, we need to know when the most recent completed event was
  var mostRecentCompletedEvent time.Time
  err = db.QueryRow("select coalesce(max(finished),'1970-01-01') from events").Scan(&mostRecentCompletedEvent)
  if err != nil {
    log.Fatalln("couldn't find most recent event", err)
    // will now exit because Fatal
  }

  
  // We like stats
  go reportProgress(*noIdleHandsFlag, status_interval, logfile)

  // We like emails for interesting things
  go sendEmails(email_interval, db, send_alerts_to, email_subject, email_header, worker)

  // set up a single goroutine to write to the db
  go reportEvent(db, worker)
  
  // Lines need to be assembled in order, so only one assembler worker to look at lines,
  // but we can at least do that on another "thread" from where we're reading input and 
  // looking for newlines. Maybe that'll make us Go Moar Faster.
  // (Now that we're using the tail package this might be a waste.)
  rawlines := make(chan *LogEvent,10000)
  go assembleRawLines(rawlines)
  
  caughtUp := false
  log.Println("Beginning to catchup scan to", mostRecentCompletedEvent)

  buffer := RawLine{}
  for {
    newEvent := LogEvent{}
    line := <- logfile.Lines
      // Now, sadly, we can't be assured that the line we're getting here is actually a complete line.
      // It appears that it might sometimes end on an EOF, and that the *next* line we read will be a continuation of the current line.
      // So buffer it, and when we find a new line, process the buffered one instead, and then buffer the current line for the next pass.
      // If our current line does *not* appear to be a new line, append to the buffer and don't process anything.
      rawline, err := decodeSyslogPrefix(line.Text)
      if err != nil {
        // Looks like this wasn't a newline after all. 
        buffer.text = fmt.Sprint(buffer.text,line.Text)
        continue
      } else {
        buffer = rawline
        if buffer.text != "" {
          newEvent.eventText = buffer.text
          newEvent.eventTimeStart = buffer.t
          newEvent.eventTimeEnd = buffer.t
          newEvent.key.host = buffer.host
          newEvent.key.pid = buffer.pid
          newEvent.key.id = buffer.keyID
          newEvent.lines = buffer.keyLine
        } else {
          // If our buffer contained nothing, do nothing
          continue
        }
      }
      
      if !caughtUp {
        lastEventAt = newEvent.eventTimeStart

        if (mostRecentCompletedEvent.Unix() < newEvent.eventTimeStart.sec) {
          log.Println("Catchup complete!")
          caughtUp = true
          rawlines <- &newEvent
        }
      } else {
          rawlines <- &newEvent
      }
  }

  // until we implement graceful exiting, we'll never get here
  AppCleanup()
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
  // see if it comes from a host we are ignoring.
  _, ignoreFromHere := ignoreTheseHosts[event.key.host]
  if(ignoreFromHere) {
    // this event comes from a host we're ignoring; forgetabouit
  } else {
    // send the text of this event to the normalizer
    var normalizeThis bytes.Buffer
    normalizeThis.WriteString(event.eventText)
    normalizeThese <- normalizeThis

    // send it into the filter chain
    firstFilter <- event
  }
}

func ignoreBlacklistedHosts(db *sql.DB) {
  ignored, err := db.Query(`select host from ignored_hosts`)
  if err != nil {
    log.Fatalln("couldn't select ignored_hosts hosts", err)
    // will now exit because Fatal
  }
  for ignored.Next() {
    var host string
    if err := ignored.Scan(&host); err != nil {
      log.Fatalln("couldn't parse ignored row", err)
      // will now exit because Fatal
    }

    ignoreTheseHosts[host] = struct{}{}
  }
  if err := ignored.Err(); err != nil {
    log.Fatalln("couldn't read ignored hosts from db", err)
    // will now exit because Fatal
  }
  ignored.Close()
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
      log.Println("shouldn't be here",killMe.id)
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

//    log.Println(time.Now(),":",flying,"in flight,",eventCount,"processed so far,",warpTo.Offset,"seek, currently at:",lastEventAt)
    log.Println(flying,"in flight,",eventCount,"processed so far,",warpTo.Offset,"seek, currently",time.Now().Unix()-lastEventAt.sec,"seconds behind")
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

func sendEmails(interval int, db *sql.DB, emails []string, subject string, header string, worker string) {

  emailHeader := "<html><body>" + header + fmt.Sprintf("\n<p>\nLast %d seconds:\n<table><tr><td>Count</td><td>Host</td><td>Normalized Event</td></tr>", interval)
  emailFooter := "</table></body></html>"

  for {    
    rows, err := db.Query(fmt.Sprintf("select count(*),host,normalize_query(event) from events where bucket_id is null and finished > now()-interval '%d seconds' and worker='%s' group by host,normalize_query(event) order by normalize_query(event),host,count(*) desc", interval, worker))
    if err != nil {
      log.Fatalln("couldn't find recent interesting events", err)
      // will now exit because Fatal
    }
    emailBody := ""
    for rows.Next() {
      var count int
      var host string
      var event string

      err = rows.Scan(&count, &host, &event)
      if err != nil {
        log.Fatalln("couldn't scan interesting event", err)
        // will now exit because Fatal
      }
      emailBody = fmt.Sprintf("%s<tr><td>%d</td><td>%s</td><td><pre>%s</pre></td></tr>\n",emailBody,count,host,event)
    }
    err = rows.Err()
    if err != nil {
      log.Fatalln("couldn't ennumerate interesting events", err)
      // will now exit because Fatal
    }
    if (!strings.EqualFold("",emailBody)) {
      for i := range emails {
        cmd := exec.Command("mailx", "-a", "Content-Type: text/html", "-s", subject, emails[i])
        cmd.Stdin = strings.NewReader(fmt.Sprintf("%s\n%s\n%s",emailHeader,emailBody,emailFooter))
        var out bytes.Buffer
        cmd.Stdout = &out
        err := cmd.Run()
        if err != nil {
          log.Fatalln(err)
          // will now exit because Fatal
        }
      }
    }

    time.Sleep(time.Duration(interval) * time.Second)
  }
}

func AppCleanup() {
  log.Println("...and that's all folks!")
  pprof.StopCPUProfile()
  if *memprofile != "" {
    f, err := os.Create(*memprofile)
    if err != nil {
      log.Fatal("could not create memory profile: ", err)
    }
    runtime.GC() // get up-to-date statistics
    if err := pprof.WriteHeapProfile(f); err != nil {
      log.Fatal("could not write memory profile: ", err)
    }
    f.Close()
  }
}
