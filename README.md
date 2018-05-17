tail-n-veil-n-mail
==================
tail-n-veil-n-mail is a project based on the very useful http://bucardo.org/wiki/Tail_n_mail.
The goal of that project is to watch your postgres logs and email you when something 
unexpected happens. It works well, but we've found that:

1. it's a performance piggy
2. once you filter out something (like, say, the footprints of a known application bug), 
   then you have zero clue on how often it's happening without additional log file digging.

tail-n-veil-n-mail attempts to address these things. We address #1 by using go instad of 
perl, and #2 by storing interesting log entries in a database for later access. On start, 
it will read the list of buckets to group log entries together into (e.g. "starting up 
noise", "bug 123", "utf8 errors") and also the regex filters used to do the grouping. 
Then it will start tailing a file and keep track of the syslog events in flight. 

When an event "completes" (either because that same syslog ID got reused, or because
enough time passed since we last saw a line for that syslog ID), the completed event is 
passed to the regex filters that got loaded from the database during startup. The event
is passed from filter to filter until one matches. When a regex filter matches a log event, 
it will be stored in the event database if the corrosponding bucket is configured to keep 
matches. If an event matches *no* filters, then it is considered "interesting", and gets
a special status in the UI.

This brings us to a serious bonus feature over tail-n-mail: because we are recording events
in a database, we can record how long they took to complete on average. If we start seeing
the "same" normalized query taking much longer than normal, we can yell about it as if it 
too is an error.

Assumptions
===========
As a young project tail-n-veil-n-mail makes a lot of assumptions. Among them:

1. You are using a centralized log server with syslog-style logging.
2. You have mailx installed where you'll be running tail-n-veil-n-mail.

tail-n-veil-n-mail actually supports multiple log servers, in independent regions...
documenting this is a future challenge.

How to use it
=============
1. Get and install Go. http://www.golang.org
2. After setting up your $GOPATH, get the code:
  go get github.com/hpcloud/tail/
  go get github.com/lib/pq
  go get github.com/benchub/tail-n-veil-n-mail
  go get github.com/lfittl/pg_query_go
3. go build github.com/benchub/tail-n-veil-n-mail
4. Import the schema. This is complicated by the fact that tnvnm supports multiple
   partitions of the data. We use it for data soverignty rules (data in the EU must 
   stay there, data in Canada must stay there, etc.) but you might also concievably need
   the partitions to deal with data volume. The point is, there are three files in 
   schema/.
   1. control.sql goes onto a "central" db. This is the one your UI, such as it is, will
      talk to. Install it into the public schema.
   2. function.sql goes into the public schema of all your dbs.
   3. partition.sql goes into a schema on just one db. You'll want to replace the string
      "tnvnm-partition-name" for each partition you push it into. Note this assumes there 
      is a user with the same name as the partition.
   4. foreign_partion.sql mirrors partition.sql, except it lives on the central server. 
      It's goal is to set up the foreign tables. You will want to modify "tnvnm-partition-name"
      and "tnvnm-server-name" as appropriate, and install in your central server for
      each partition.
   5. In addition to all that, the roles "tnvnm" and "www" are expected to be on every 
      server, and also reachable to every server from the central db as foreign servers.
      Remember to create the user mappings, if needed, for any foreign servers.
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
   4. buckets.active=true means the bucket should be displayed by the UI by default. This
      is for cases where a bucket is expected to be matched, but you really don't care
      about the matches. If you really want to see them, you could, but by and large,
      things that would match inactive buckets are out of mind.
   5. filters.report=true means that filters.uses will increment when a filter matches.
   Finally, given that this is a replacement for tail-n-mail, you might (wrongly) assume
   that the perl regular expressions you wrote for tail-n-mail will work here. They may
   or may not - Go's regular expressions are a little more Ivory colored than perl's.
   Also, while tail-n-mail squashed everything on to one line (most of the time),
   tail-n-veil-n-mail exlicitly only does that when there's a newline in the query. You
   may need to preface your regex patterns with (?s) to get them to span newlines.
7. Modify the conf to fit your environment.
   1. DBConn is hopefully self-explanitory
   2. StatusInterval is how often to report status (in seconds).
   3. EmailInterval is how often to look for interesting things (in seconds).
   4. EmailsTo is an array of emails to send a notice to if any intersting things have
      observed since the last check.
8. Run it already, like so: tail_n_veil_n_mail -config=conf -log=/var/log/postgres.log
   If you're recovering from a crash, -warp will be useful to not have to restart at the
   beginning. Just use the most recent seek value from the output (assuming the file 
   hasn't wrapped.)
9. Point a web browser at the php files in web/. Yes, a better UI would be better.

Known Issues
============
It's not yet done

TODO
====
Um yeah quite a bit.

- Support more than syslog postgres logs
- Configurable log prefixes
- Use partitioning for the events table
- Much moar better web interface
- Support host name FQDNs so that we can distinguish between two "db1" instances in the
  same event db.
- Allow run-time config reloading. We sort of have this, if the filter config changes,
  but not if we want to make conf file changes only.
- Allow observations of a bucket to also trigger emails, instead of just interesting things.
  Probably good to give each bucket a specific list of email addresses to use if the
  default isn't wanted.
- Support a MUA other than mailx.
- Configurable alert email.
- Allow buckets to apply to host groups, intead of just specific hosts.

