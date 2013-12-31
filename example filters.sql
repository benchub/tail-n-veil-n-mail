COPY buckets (id, name, eat_it, report_it, workers, active) FROM stdin;
2       starting up     t       f       1       t
3       wal streaming when starting up  t       f       1       t
4       admin query kills       t       f       1       t
5       shutdown        t       f       1       t
6       queries by real people  t       f       1       t
11      missing PG_VERSION      t       f       1       t
110     bad utf8        t       t       1       t
111     die autovacuum die      t       t       1       t
114     zombie transactions     t       t       1       t
121     non-errors      t       f       2       t
122     checkpoint start        t       f       1       t
123     checkpoint complete     t       f       1       t
124     reach restart point     t       f       1       t
125     temp file needed        t       t       1       t
126     no exclusive lock for autovacuum        t       t       1       t
127     deadlock        t       t       1       t
128     tcp pings       t       f       1       t
130     backup  t       f       1       t
131     pgstat timeout  t       t       1       t
132     queries on slave ran too long   t       t       1       t
\.

SELECT pg_catalog.setval('buckets_id_seq', 140, true);


COPY filters (bucket_id, filter, id, uses, report) FROM stdin;
2       db=,user= LOG:  redo starts at  2       89      t
2       database system is ready to accept connections  150     24      t
2       db=,user= LOG:  database system was interrupted; last known up at       1       7       t
2       db=,user= LOG:  creating missing WAL directory "pg_xlog/archive_status" 140     7       t
2       db=,user= LOG:  database system is ready to accept read only connections        3       82      t
2       db=,user= LOG:  entering standby mode   134     94      t
2       db=,user= LOG:  consistent recovery state reached at    139     84      t
2       db=,user= LOG:  restored log file "[^"]+" from archive  133     328743  t
2       db=,user= LOG:  autovacuum launcher started     154     24      t
2       the database system is starting up      4       6855    t
2       db=,user= LOG:  unexpected pageaddr [^/]+/[^ ]+ in log file [0-9]+, segment [0-9]+, offset [0-9]+       156     46      t
3       could not (send|receive) data (from|to) WAL stream:     5       216     t
3       db=,user= LOG:  streaming replication successfully connected to primary 144     303     t
4       terminating connection due to administrator command     6       299     t
5       db=,user= LOG:  aborting any active transactions        147     87      t
5       terminating walreceiver process due to administrator command    7       46      t
5       db=,user= LOG:  database system was shut down at        151     20      t
5       db=,user= LOG:  shutting down   69      88      t
5       db=,user= LOG:  autovacuum launcher shutting down       146     22      t
5       db=postgres,user=monitor FATAL:  the database system is shutting down   149     32      t
5       db=,user= FATAL:  terminating autovacuum process due to administrator command   153     301     t
5       db=,user= LOG:  database system is shut down    148     82      t
5       db=,user= LOG:  received fast shutdown request  145     91      t
6       db=[^,]+,user=[^-]+-r(o|w)      8       169681  t
11      File "/var/lib/postgresql/9.1/main/PG_VERSION" is missing       13      0       t
110     invalid byte sequence for encoding "UTF8":      120     0       t
111     canceling autovacuum task       121     20      t
114     current transaction is aborted, commands ignored until end of transaction       124     105     t
121     db=[^,]+,user=[^ ]+ LOG:  duration      132     0       f
122     db=,user= LOG:  (check|restart)point starting:  142     114957  t
123     db=,user= LOG:  (restart|check)point complete:  143     114910  t
124     db=,user= LOG:  recovery restart point  135     75190   t
125     LOG:  temporary file: path "base/pgsql_tmp/pgsql_tmp    136     121975  t
126     db=,user= LOG:  automatic vacuum of table ".+": could not \\(re\\)acquire exclusive lock for truncate scan      137     1240    t
127     ERROR:  deadlock detected       138     597     t
128     db=\\[unknown\\],user=\\[unknown\\] LOG:  incomplete startup packet     141     181     t
130     db=,user= LOG:  recovery has paused     129     418     t
131     WARNING:  pgstat wait timeout   152     38      t
132     ERROR:  canceling statement due to conflict with recovery       155     6       t
\.



SELECT pg_catalog.setval('filters_id_seq', 165, true);

