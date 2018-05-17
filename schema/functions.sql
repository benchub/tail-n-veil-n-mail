-- triggers for config changes

CREATE FUNCTION buckets_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
  PERFORM pg_notify('configChange','foo');
  RETURN NEW;
END;
$$;
ALTER FUNCTION public.buckets_update() OWNER TO tnvnm;

CREATE FUNCTION filters_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
  if (new.uses = 0) or (new.filter != old.filter) or (new.bucket_id != old.bucket_id) or (new.report != old.report)
  then
    PERFORM pg_notify('configChange','foo');
  end if;
  RETURN NEW;
END;
$$;
ALTER FUNCTION public.filters_update() OWNER TO tnvnm;

CREATE FUNCTION merge_buckets(a_id integer, b_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $_$
  DECLARE
    s text;
  BEGIN
    for s in select schema from data_sources
    loop
      execute 'update "' || s || '".events set bucket_id=$1 where bucket_id=$2' using b_id,a_id;
      execute 'delete from "' || s || '".filters where bucket_id=$1' using a_id;
      execute 'delete from "' || s || '".onlyon where bucket_id=$1' using a_id;
      execute 'delete from "' || s || '".buckets where id=$1' using a_id;
      execute 'update "' || s || '".buckets set bucket_id=$1 where bucket_id=$1' using b_id;
    end loop;
  END;
$_$;
ALTER FUNCTION public.merge_buckets(a_id integer, b_id integer) OWNER TO tnvnm;

CREATE FUNCTION onlyon_update() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
BEGIN
  PERFORM pg_notify('configChange','foo');
  RETURN NEW;
END;
$$;
ALTER FUNCTION public.onlyon_update() OWNER TO tnvnm;


-- functions for reporting

CREATE FUNCTION normalize_query(text, OUT text) RETURNS text
    LANGUAGE sql
    AS $_$
  SELECT
    regexp_replace(regexp_replace(regexp_replace(regexp_replace(
    regexp_replace(regexp_replace(regexp_replace(regexp_replace(
    regexp_replace(regexp_replace(

    lower($1),

    -- Remove our db syslog line headers
    -- A great spot to allow for more flexibility in the future
    'db=[^,]*,user=[^ ]* ',        '',            'g'   ),

    -- Remove extra space, new line and tab caracters by a single space
    '\\s+',                        ' ',           'g'   ),

    -- Remove string content
    $$\\'$$,                       '',            'g'   ),
    $$'[^']*'$$,                   $$''$$,        'g'   ),
    $$''('')+$$,                   $$''$$,        'g'   ),

    -- Remove NULL parameters
    '= *NULL',                     '=0',          'g'   ),

    -- Remove comments
    '/\\*.*\\*/',                  '/* */',       'g'   ),

    -- Remove numbers
    '[0-9]+',                       '0',     'g'   ),

    -- Remove hexadecimal numbers
    '([^a-z_$-])0x[0-9a-f]{1,10}', '\1'||'0x',    'g'   ),

    -- Remove IN values
    ' in *\\([''0x,\\s]*\\)',      ' in (...)',   'g'   )
  ;
$_$;
ALTER FUNCTION public.normalize_query(text, OUT text) OWNER TO tnvnm;


-- functions for management
-- To distribute changes, we need to write the same change to all partitions
-- This can be annoying when there are multiple partitions (and especially
-- when those partitions are spread out across the world) so these functions
-- exist to ease the pain.

CREATE FUNCTION set_bucket_activation(b_id integer, v boolean) RETURNS void
    LANGUAGE plpgsql
    AS $_$
  DECLARE
    s text;
  BEGIN
    for s in select schema from data_sources
    loop
      execute 'update "' || s || '".buckets set active=$1 where id=$2' using v,b_id;
      raise notice 'processing schema %s', s;
    end loop;
  END;
$_$;
ALTER FUNCTION public.set_bucket_activation(b_id integer, v boolean) OWNER TO tnvnm;


CREATE FUNCTION set_bucket_name(b_id integer, v text) RETURNS void
    LANGUAGE plpgsql
    AS $_$
  DECLARE
    s text;
  BEGIN
    for s in select schema from data_sources
    loop
      execute 'update "' || s || '".buckets set name=$1 where id=$2' using v,b_id;
      raise notice 'processing schema %s', s;
    end loop;
  END;
$_$;
ALTER FUNCTION public.set_bucket_name(b_id integer, v text) OWNER TO tnvnm;


CREATE FUNCTION set_bucket_reporting(b_id integer, v boolean) RETURNS void
    LANGUAGE plpgsql
    AS $_$
  DECLARE
    s text;
  BEGIN
    for s in select schema from data_sources
    loop
      execute 'update "' || s || '".buckets set report_it=$1 where id=$2' using v,b_id;
      raise notice 'processing schema %s', s;
    end loop;
  END;
$_$;
ALTER FUNCTION public.set_bucket_reporting(b_id integer, v boolean) OWNER TO tnvnm;


CREATE FUNCTION start_ignoring_host(h text) RETURNS void
    LANGUAGE plpgsql
    AS $_$
DECLARE
    s text;
BEGIN
    for s in select schema from data_sources
    loop
      execute 'insert into "' || s || '".ignored_hosts (host) values ($1)' using h;
      raise notice 'processing schema %s', s;
    end loop;
END;
$_$;
ALTER FUNCTION public.start_ignoring_host(h text) OWNER TO tnvnm;


CREATE FUNCTION stop_ignoring_host(h text) RETURNS void
    LANGUAGE plpgsql
    AS $_$
DECLARE
    s text;
BEGIN
    for s in select schema from data_sources
    loop
      execute 'delete from "' || s || '".ignored_hosts where host=$1' using h;
      raise notice 'processing schema %s', s;
    end loop;
END;
$_$;
ALTER FUNCTION public.stop_ignoring_host(h text) OWNER TO tnvnm;
