CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;
COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';

CREATE FUNCTION normalize_query(text, OUT text) RETURNS text
    LANGUAGE sql
    AS $_$
  SELECT
    regexp_replace(regexp_replace(regexp_replace(regexp_replace(
    regexp_replace(regexp_replace(regexp_replace(regexp_replace(
    regexp_replace(
    regexp_replace(

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
    '([^a-z_$-])-?([0-9]+)',       '\1'||'0',     'g'   ),

    -- Remove hexadecimal numbers
    '([^a-z_$-])0x[0-9a-f]{1,10}', '\1'||'0x',    'g'   ),

    -- Remove IN values
    ' in *\\([''0x,\\s]*\\)',      ' in (...)',   'g'   )
  ;
$_$;
ALTER FUNCTION public.normalize_query(text, OUT text) OWNER TO tnvnm;


CREATE TABLE buckets (
    id integer NOT NULL,
    name text,
    eat_it boolean DEFAULT true NOT NULL,
    report_it boolean DEFAULT true NOT NULL,
    workers integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL
);
ALTER TABLE public.buckets OWNER TO tnvnm;

CREATE SEQUENCE buckets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.buckets_id_seq OWNER TO tnvnm;
ALTER SEQUENCE buckets_id_seq OWNED BY buckets.id;


CREATE TABLE events (
    bucket_id integer,
    event text NOT NULL,
    started timestamp with time zone NOT NULL,
    finished timestamp with time zone NOT NULL,
    lines integer NOT NULL,
    fragment boolean DEFAULT false NOT NULL,
    host text NOT NULL
);
ALTER TABLE public.events OWNER TO tnvnm;


CREATE TABLE filters (
    bucket_id integer NOT NULL,
    filter text,
    id integer NOT NULL,
    uses integer DEFAULT 0 NOT NULL,
    report boolean DEFAULT true NOT NULL
);
ALTER TABLE public.filters OWNER TO tnvnm;

CREATE SEQUENCE filters_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.filters_id_seq OWNER TO tnvnm;
ALTER SEQUENCE filters_id_seq OWNED BY filters.id;


CREATE TABLE onlyon (
    bucket_id integer NOT NULL,
    host text NOT NULL
);
ALTER TABLE public.onlyon OWNER TO tnvnm;

ALTER TABLE ONLY buckets ALTER COLUMN id SET DEFAULT nextval('buckets_id_seq'::regclass);
ALTER TABLE ONLY filters ALTER COLUMN id SET DEFAULT nextval('filters_id_seq'::regclass);


ALTER TABLE ONLY buckets
    ADD CONSTRAINT buckets_pkey PRIMARY KEY (id);


ALTER TABLE ONLY filters
    ADD CONSTRAINT filters_filter_key UNIQUE (filter);


ALTER TABLE ONLY filters
    ADD CONSTRAINT filters_pkey PRIMARY KEY (id);

ALTER TABLE ONLY onlyon
    ADD CONSTRAINT onlyon_pkey PRIMARY KEY (bucket_id, host);

CREATE UNIQUE INDEX buckets_name_key ON buckets USING btree (name);

CREATE INDEX event_what_and_when ON events USING btree (bucket_id, finished);



ALTER TABLE ONLY events
    ADD CONSTRAINT events_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


ALTER TABLE ONLY filters
    ADD CONSTRAINT filters_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


ALTER TABLE ONLY onlyon
    ADD CONSTRAINT onlyon_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM tnvnm;
GRANT ALL ON SCHEMA public TO tnvnm;
GRANT ALL ON SCHEMA public TO PUBLIC;



REVOKE ALL ON TABLE buckets FROM PUBLIC;
REVOKE ALL ON TABLE buckets FROM tnvnm;
GRANT ALL ON TABLE buckets TO tnvnm;
GRANT SELECT ON TABLE buckets TO www;



REVOKE ALL ON TABLE events FROM PUBLIC;
REVOKE ALL ON TABLE events FROM tnvnm;
GRANT ALL ON TABLE events TO tnvnm;
GRANT SELECT ON TABLE events TO www;


