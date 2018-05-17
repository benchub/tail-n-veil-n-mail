package main

import (
  "strings"
  "regexp"
  "log"
  "bytes"
  "time"
  "sync"
  "strconv"
  "math"
  "database/sql"

  "github.com/lfittl/pg_query_go"
)

type RunningStat struct {
	m_n int64
	m_oldM float64
	m_newM float64
	m_oldS float64
	m_newS float64
}

type Samples struct {
	// how many ms
	duration float64

	// when
	unixtime int64
}

type Fingerprint struct {
  statsLock sync.RWMutex
  fingerprint string
  normalized string
  samples chan Samples
  count int64
  last int64
  sum float64
  stats RunningStat
  db_id int64
}

// A dictionary of what fingerprints we've seen since startup and are currently processing
var protectedFingerprints = struct{
  sync.RWMutex
  m map[string]*Fingerprint
} {m: make(map[string]*Fingerprint)}


func normalizeEvents(db *sql.DB, c chan bytes.Buffer) {
  var isActualQuery,_ = regexp.Compile(`(?s)LOG:\s+duration:\s+(\d+\.\d+)\s+ms\s+(execute|statement)[^:]*:((.+)+(/\*.+\*/)|(.+)+)`)

  for {
    text := <-c
    
    matched := isActualQuery.MatchString(text.String())
    
    if matched {
      // we want the second match group, which is the actual query.
      matches := isActualQuery.FindStringSubmatch(text.String())
      raw_sql := matches[4]
      if raw_sql == "" {
      	raw_sql = matches[3]
      }
      clean_sql := strings.Replace(raw_sql,"#011", "  ", -1)
      
      // get rid of lines starting with --

      single_line_sql := strings.Replace(clean_sql,"\n","",-1)
      duration := matches[1]

      normalized, err := pg_query.Normalize(single_line_sql)
      if err != nil {
      	log.Printf("couldn't normalize %s because %s", single_line_sql, err)
      	continue
      }
      fingerprint, err := pg_query.FastFingerprint(single_line_sql)
      if err != nil {
      	log.Printf("couldn't normalize %s RAW QUERY (%s) because %s", single_line_sql, raw_sql, err)
      	continue
      }

      sample := Samples{}
      sample.unixtime = time.Now().Unix()
      sample.duration, err = strconv.ParseFloat(duration,64)
      if err != nil {
      	log.Printf("Couldn't conver duration %s to float, because %s", duration, err)
      	continue
      }

      if (false) {
	      log.Printf("sql %s took %s ms, hashes to %s and normalizes to %s", text.String(), duration, fingerprint, normalized)
	  }

	  // If we've already started a goroutine for this fingerprint, send this event to that channel.
	  // If not, start a new goroutine and make a channel for it to consume from.
      protectedFingerprints.RLock()
	  existingFingerprint, present := protectedFingerprints.m[fingerprint]
	  protectedFingerprints.RUnlock()
	  if present {
	  	existingFingerprint.samples <- sample
	  } else {
		newFingerprint := Fingerprint{}
		var db_id int64

	    // figure out which db_id this fingerprint will use. It will never change, so just cache it at creation
		err := db.QueryRow(`select id from fingerprints where fingerprint=$1`, fingerprint).Scan(&db_id)
		switch {
			case err == sql.ErrNoRows:
				// looks like we have a new fingerprint
				err := db.QueryRow(`insert into fingerprints (fingerprint,normalized) values ($1,$2) returning id`, fingerprint, normalized).Scan(&db_id)
				switch {
					case err == sql.ErrNoRows:
						log.Println("couldn't insert returning for fingerprint", fingerprint, err)
						continue
					case err != nil:
						log.Println("couldn't insert fingerprint", fingerprint, normalized, err)
						continue
					default:
						newFingerprint.db_id = db_id
				}
			case err != nil:
				log.Fatalln("couldn't select fingerprint id for", fingerprint, err)
				// will now exit because Fatal
			default:
				newFingerprint.db_id = db_id
		}

		newFingerprint.samples = make(chan Samples)
		newFingerprint.fingerprint = fingerprint
		newFingerprint.normalized = normalized
	    
	    protectedFingerprints.Lock()
	    protectedFingerprints.m[fingerprint] = &newFingerprint
	    protectedFingerprints.Unlock()

	    go consumeSamples(&newFingerprint)
	    go reportSamples(db, &newFingerprint)
   	  }
    }
  }
}

func reportSamples(db *sql.DB, f *Fingerprint) {
	lastReport := f.last
	for {
		time.Sleep(60*time.Second)
		if f.last > lastReport {
			var dbStats RunningStat
			var combined RunningStat

			tx, err := db.Begin();
			if err != nil {
				log.Println("couldn't start transaction for", f.db_id,err)
				continue
			}
			// lock the stats block for reading
			f.statsLock.RLock()
			err = db.QueryRow(`select count,mean,deviation from fingerprint_stats where fingerprint_id=$1 for update`, f.db_id).Scan(&dbStats.m_n,&dbStats.m_oldM,&dbStats.m_oldS)
			switch {
				case err == sql.ErrNoRows:
				    r, err := db.Query(`INSERT INTO fingerprint_stats(fingerprint_id, last, count, mean, deviation) VALUES ($1, $2, $3, $4, $5)`, f.db_id, f.last, f.stats.m_n, f.stats.m_oldM, RunningStatDeviation(f.stats))
				    if err != nil {
				      log.Println("couldn't insert new fingerprint stats for fingerprint", f.db_id, err)
				      f.statsLock.RUnlock()
				      tx.Rollback()
				      continue
				    }
				    r.Close()
					// lock the stats block for writing; these stats are in the db; we don't need to keep counting them.
					f.statsLock.RUnlock()
					f.statsLock.Lock()
					f.stats.m_n = 0
					f.stats.m_oldM = 0
					f.stats.m_newM = 0
					f.stats.m_oldS = 0
					f.stats.m_newS = 0
					f.statsLock.Unlock()
					f.statsLock.RLock()
			     case err != nil:
			     	log.Println("couldn't retrieve existing fingerprint stats", err)
			        f.statsLock.RUnlock()
			        tx.Rollback()
			        continue
			     default:
			     	// we had stats before; merge them with what we have now, then zero out what we have so we only merge in new data
					// https://gist.github.com/turnersr/11390535
					dbStats.m_newM = dbStats.m_oldM
					dbStats.m_newS = dbStats.m_oldS

					delta := dbStats.m_oldM - f.stats.m_oldM
					delta2 := delta*delta 
					combined.m_n = f.stats.m_n + dbStats.m_n
					combined.m_oldM = f.stats.m_newM + float64(dbStats.m_n)*delta/float64(combined.m_n)
					combined.m_newM = combined.m_oldM

					q := float64(f.stats.m_n * dbStats.m_n) * delta2 / float64(combined.m_n)
					combined.m_oldS = f.stats.m_newS + dbStats.m_newS + q
					combined.m_newS = combined.m_oldS

					// lock the stats block for writing
					f.statsLock.RUnlock()
					f.statsLock.Lock()
					f.stats.m_n = 0
					f.stats.m_oldM = 0
					f.stats.m_newM = 0
					f.stats.m_oldS = 0
					f.stats.m_newS = 0
					f.statsLock.Unlock()
					f.statsLock.RLock()

					r, err := db.Query(`update fingerprint_stats set last=$1,count=$2,mean=$3,deviation=$4 where fingerprint_id=$5`,f.last,combined.m_n,combined.m_oldM,combined.m_oldS,f.db_id)
			        if err != nil {
				        log.Println("couldn't update fingerprint stats for fingerprint", f.db_id, err)
				        f.statsLock.RUnlock()
				        tx.Rollback()
				        continue
				    }
				    r.Close()
					log.Printf("fingerprint %d has seen %d calls; last at %d, sum at %f (%f), mean %f, deviation %f", f.db_id, combined.m_n, f.last, f.sum, float64(combined.m_n)*combined.m_oldM, RunningStatMean(combined), RunningStatDeviation(combined))
			}

			err = tx.Commit()
			if err != nil {
				log.Printf("Couldn't commit fingerprint stats update for",f.db_id)
			}
			lastReport = f.last
			f.statsLock.RUnlock()
		}
		// it would be slick if we got rid of this fingerprint if it didn't happen again for a while
	}
}

func consumeSamples(f *Fingerprint) {

	for {
		sample := <- f.samples

		// lock the stats block for writing
		f.statsLock.Lock()
		f.last = sample.unixtime
		f.sum += sample.duration
		f.stats = Push(sample.duration,f.stats)
		f.statsLock.Unlock()
	}
}

// https://www.johndcook.com/blog/standard_deviation/
func Push(x float64, oldRS RunningStat) RunningStat {
	rs := oldRS
	rs.m_n += 1 
	if(rs.m_n == 1) {
		rs.m_oldM = x
		rs.m_newM = x
		rs.m_oldS = 0
	} else {
		rs.m_newM = rs.m_oldM + (x - rs.m_oldM)/float64(rs.m_n)
		rs.m_newS = rs.m_oldS + (x - rs.m_oldM)*(x - rs.m_newM)

		rs.m_oldM = rs.m_newM
		rs.m_oldS = rs.m_newS
	}
	return rs
}

func RunningStatMean(rs RunningStat) float64 {
	if rs.m_n > 0 {
		return rs.m_newM
	}

	return 0
}

func RunningStatVariance(rs RunningStat) float64 {
	if rs.m_n > 1 {
		return rs.m_newS/float64(rs.m_n - 1)
	}

	return 0
}

func RunningStatDeviation(rs RunningStat) float64 {
	return math.Sqrt(RunningStatVariance(rs))
}
