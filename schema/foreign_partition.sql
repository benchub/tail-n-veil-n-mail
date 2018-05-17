create schema "tnvnm-partition-name";
grant usage on schema "tnvnm-partition-name" to www;

create foreign table "tnvnm-partition-name".buckets (
    id integer NOT NULL,
    name text,
    eat_it boolean DEFAULT true NOT NULL,
    report_it boolean DEFAULT true NOT NULL,
    workers integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL
) server "tnvnm-server-name" ;

create foreign table "tnvnm-partition-name".ignored_hosts (
    host text unique not null
) server "tnvnm-server-name" ;

create foreign table "tnvnm-partition-name".fingerprints (
    id integer NOT NULL,
    fingerprint text NOT NULL,
    normalized text NOT NULL
) server "tnvnm-server-name" ;

create foreign table "tnvnm-partition-name".fingerprint_stats (
    fingerprint_id bigint NOT NULL,
    count bigint,
    mean double precision,
    deviation double precision,
    last integer
) server "tnvnm-server-name" ;

create foreign table "tnvnm-partition-name".filters (
    bucket_id integer NOT NULL,
    filter text,
    id integer NOT NULL,
    uses integer DEFAULT 0 NOT NULL,
    report boolean DEFAULT true NOT NULL
) server "tnvnm-server-name" ;

create foreign table "tnvnm-partition-name".onlyon (
    bucket_id integer NOT NULL,
    host text NOT NULL
) server "tnvnm-server-name" ;

create foreign table "tnvnm-partition-name".events  (
    bucket_id integer,
    event text NOT NULL,
    started timestamp with time zone NOT NULL,
    finished timestamp with time zone NOT NULL,
    lines integer NOT NULL,
    fragment boolean DEFAULT false NOT NULL,
    host text NOT NULL,
    id integer not null
) server "tnvnm-server-name" ;

grant select on "tnvnm-partition-name".buckets to www;
grant select on "tnvnm-partition-name".events to www;
grant insert,update,delete on "tnvnm-partition-name".buckets to tnvnm;
grant insert,update,delete on "tnvnm-partition-name".onlyon to tnvnm;
grant insert,update,delete on "tnvnm-partition-name".filters to tnvnm;