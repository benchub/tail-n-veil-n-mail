tail-n-veil-n-mail
==================
tail-n-veil-n-mail is a project based on the very useful http://bucardo.org/wiki/Tail_n_mail.
The goal of that project is to watch your postgres logs and email you when something 
unexpected happens. It works well, but we've found that:

1. it's a performance piggy
2. once you filter out something (like, say, the footprints of a known application bug), 
   then you have zero clue on how often it's happening without additional log file digging.

tail-n-veil-n-mail attempts to address these things. We address #1 by using go instad of 
perl, and #2 by storing filtered log entries in a database. On start, it will read the 
list of buckets to group log entries together into (e.g. "starting up noise", "bug 123", 
"utf8 errors") and the regex filters used to do the grouping. Then it will start tailing a
file and do a whole lot of regex matching. When a regex filter matches a log event, it 
will be stored in the event database if the corrosponding bucket is configured to keep 
matches. 

Assumptions
===========
As a young project tail-n-veil-n-mail makes a lot of assumptions. Among them:

1. You are using a centralized log server with syslog-style logging.
2. You have databases with distinct non-qualified host names (i.e. db1 and db2, *not* 
   db.foo and db.bar)

How to use it
=============
1. Get and install Go. http://www.golang.org
2. After setting up your $GOPATH, get the code:
  go get github.com/ActiveState/tail/
  go get github.com/lib/pq
  go get github.com/benchub/tail-n-veil-n-mail
3. go build github.com/benchub/tail-n-veil-n-mail
4. Get yourself a postgres database somewhere and import schema.sql into it. Note that it
   assumes you have a role called tnvnm and another called www. It would be good to make 
   these roles if you don't have them.
6. Import some buckets and their associated filters. example filters.sql has some examples.
   Also see https://github.com/benchub/tail-n-veil-n-mail-import, which makes it easy. 
   Some things to know:
   1. buckets.eat_it=true means that a match with this bucket won't be passed on down to
      the next filter.
   2. buckets.report_it=true means that matches to the bucket will be recorded in the db.
   3. buckets.workers is a count for how many concurrent works will be looking for matches
      for the filters for this bucket. Usually you want this to be 1, but if you have 
      something that will match far more often than other things, give it another worker.
      The speed may or may not make much of a difference, but the filter chain is built 
      with buckets having the most workers doing their work first (the idea being there
      will be less work for the remaining buckets).
   4. buckets.active=true means the bucket will get loaded by tail-n-veil-n-mail upon 
      start.
   5. filters.report=true means that filters.uses will increment when a filter matches.
7. Modify conf to fit your environment.
   1. DBConn is hopefully self-explanitory
   2. StatusInterval is how often to report status to stdout.
8. Run it already, like so: tail_n_veil_n_mail -config=conf -log=/var/log/postgres.log
   If you're recovering from a crash, -warp will be useful to not have to restart at the
   beginning. Just use the most recent seek value from the output (assuming the file 
   hasn't wrapped.)
9. Point a web browser at the php files in web/. Yes, a better UI would be better.

Known Issues
============
When tailing for new log lines, we wait for a full line before processing. Or at least
we're supposed to only get full lines; sometimes, it seems we get partial lines. That 
throws off our parsers and we crash. So lame. We need to buffer lines so that if the
current line appears to be a genuine new line, only then process the previous line. 
Otherwise, append the current line to the previous line and hope for the best with the
next line we read.

TODO
====
Um yeah quite a bit.

- Support more than syslog postgres logs
- Configurable log prefixes
- Use partitioning for the events table
- Much moar better web interface
- Support host name FQDNs so that we can distinguish between two "db1" instances in the
  same event db.
- Allow run-time config reloading.
- Dump status to a logfile instead of stdout.
- Dump errors to a logfile instead of stderr.
