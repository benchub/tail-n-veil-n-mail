--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- Name: tnvnm-partition-name; Type: SCHEMA; Schema: -; Owner: tnvnm
--

CREATE SCHEMA "tnvnm-partition-name";


ALTER SCHEMA "tnvnm-partition-name" OWNER TO tnvnm;

SET search_path = "tnvnm-partition-name", pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: buckets; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE buckets (
    id integer NOT NULL,
    name text,
    eat_it boolean DEFAULT true NOT NULL,
    report_it boolean DEFAULT true NOT NULL,
    workers integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL
);


ALTER TABLE "tnvnm-partition-name".buckets OWNER TO tnvnm;

--
-- Name: buckets_id_seq; Type: SEQUENCE; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE SEQUENCE buckets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE "tnvnm-partition-name".buckets_id_seq OWNER TO tnvnm;

--
-- Name: buckets_id_seq; Type: SEQUENCE OWNED BY; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER SEQUENCE buckets_id_seq OWNED BY buckets.id;


--
-- Name: events; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE events (
    bucket_id integer,
    id integer NOT NULL,
    event text NOT NULL,
    started timestamp with time zone NOT NULL,
    finished timestamp with time zone NOT NULL,
    lines integer NOT NULL,
    fragment boolean DEFAULT false NOT NULL,
    host text NOT NULL,
    worker text NOT NULL
);


ALTER TABLE "tnvnm-partition-name".events OWNER TO tnvnm;

--
-- Name: events_id_seq; Type: SEQUENCE; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE SEQUENCE events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE "tnvnm-partition-name".events_id_seq OWNER TO tnvnm;

--
-- Name: events_id_seq; Type: SEQUENCE OWNED BY; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER SEQUENCE events_id_seq OWNED BY events.id;


--
-- Name: filters; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE filters (
    bucket_id integer NOT NULL,
    filter text,
    id integer NOT NULL,
    uses bigint DEFAULT 0 NOT NULL,
    report boolean DEFAULT true NOT NULL
);


ALTER TABLE "tnvnm-partition-name".filters OWNER TO tnvnm;

--
-- Name: filters_id_seq; Type: SEQUENCE; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE SEQUENCE filters_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE "tnvnm-partition-name".filters_id_seq OWNER TO tnvnm;

--
-- Name: filters_id_seq; Type: SEQUENCE OWNED BY; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER SEQUENCE filters_id_seq OWNED BY filters.id;


--
-- Name: fingerprint_stats; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE fingerprint_stats (
    fingerprint_id bigint NOT NULL,
    count bigint,
    mean double precision,
    deviation double precision,
    last integer
);


ALTER TABLE "tnvnm-partition-name".fingerprint_stats OWNER TO tnvnm;

--
-- Name: fingerprints; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE fingerprints (
    id integer NOT NULL,
    fingerprint text NOT NULL,
    normalized text NOT NULL
);


ALTER TABLE "tnvnm-partition-name".fingerprints OWNER TO tnvnm;

--
-- Name: fingerprints_id_seq; Type: SEQUENCE; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE SEQUENCE fingerprints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE "tnvnm-partition-name".fingerprints_id_seq OWNER TO tnvnm;

--
-- Name: fingerprints_id_seq; Type: SEQUENCE OWNED BY; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER SEQUENCE fingerprints_id_seq OWNED BY fingerprints.id;


--
-- Name: ignored_hosts; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE ignored_hosts (
    host text NOT NULL
);


ALTER TABLE "tnvnm-partition-name".ignored_hosts OWNER TO tnvnm;

--
-- Name: onlyon; Type: TABLE; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE TABLE onlyon (
    bucket_id integer NOT NULL,
    host text NOT NULL
);


ALTER TABLE "tnvnm-partition-name".onlyon OWNER TO tnvnm;

--
-- Name: id; Type: DEFAULT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY buckets ALTER COLUMN id SET DEFAULT nextval('buckets_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY events ALTER COLUMN id SET DEFAULT nextval('events_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY filters ALTER COLUMN id SET DEFAULT nextval('filters_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY fingerprints ALTER COLUMN id SET DEFAULT nextval('fingerprints_id_seq'::regclass);


--
-- Name: buckets_pkey; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY buckets
    ADD CONSTRAINT buckets_pkey PRIMARY KEY (id);


--
-- Name: filters_filter_key; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY filters
    ADD CONSTRAINT filters_filter_key UNIQUE (filter);


--
-- Name: filters_pkey; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY filters
    ADD CONSTRAINT filters_pkey PRIMARY KEY (id);


--
-- Name: fingerprint_stats_pkey; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY fingerprint_stats
    ADD CONSTRAINT fingerprint_stats_pkey PRIMARY KEY (fingerprint_id);


--
-- Name: fingerprints_unique; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY fingerprints
    ADD CONSTRAINT fingerprints_unique UNIQUE (fingerprint);


--
-- Name: ignored_hosts_host_key; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY ignored_hosts
    ADD CONSTRAINT ignored_hosts_host_key UNIQUE (host);


--
-- Name: onlyon_pkey; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY onlyon
    ADD CONSTRAINT onlyon_pkey PRIMARY KEY (bucket_id, host);


--
-- Name: tnvnm-partition-name-events_pkey; Type: CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

ALTER TABLE ONLY events
    ADD CONSTRAINT "tnvnm-partition-name-events_pkey" PRIMARY KEY (id);


--
-- Name: buckets_name_key; Type: INDEX; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE UNIQUE INDEX buckets_name_key ON buckets USING btree (name);


--
-- Name: fingerprints_id_idx; Type: INDEX; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE UNIQUE INDEX fingerprints_id_idx ON fingerprints USING btree (id);


--
-- Name: tnvnm-partition-name-event_what_and_when; Type: INDEX; Schema: tnvnm-partition-name; Owner: tnvnm; Tablespace:
--

CREATE INDEX "tnvnm-partition-name-event_what_and_when" ON events USING btree (bucket_id, finished);


--
-- Name: notify_config_change_on_buckets; Type: TRIGGER; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE TRIGGER notify_config_change_on_buckets AFTER INSERT OR UPDATE ON buckets FOR EACH ROW EXECUTE PROCEDURE public.buckets_update();


--
-- Name: notify_config_change_on_filters; Type: TRIGGER; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE TRIGGER notify_config_change_on_filters AFTER INSERT OR UPDATE ON filters FOR EACH ROW EXECUTE PROCEDURE public.filters_update();


--
-- Name: notify_config_change_on_onlyon; Type: TRIGGER; Schema: tnvnm-partition-name; Owner: tnvnm
--

CREATE TRIGGER notify_config_change_on_onlyon AFTER INSERT OR UPDATE ON onlyon FOR EACH ROW EXECUTE PROCEDURE public.onlyon_update();


--
-- Name: filters_bucket_id_fkey; Type: FK CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY filters
    ADD CONSTRAINT filters_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


--
-- Name: fingerprint_stats_fingerprint_id_fkey; Type: FK CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY fingerprint_stats
    ADD CONSTRAINT fingerprint_stats_fingerprint_id_fkey FOREIGN KEY (fingerprint_id) REFERENCES fingerprints(id);


--
-- Name: onlyon_bucket_id_fkey; Type: FK CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY onlyon
    ADD CONSTRAINT onlyon_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


--
-- Name: tnvnm-partition-name_events.bucket_id_fkey; Type: FK CONSTRAINT; Schema: tnvnm-partition-name; Owner: tnvnm
--

ALTER TABLE ONLY events
    ADD CONSTRAINT events_bucket_id_fkey FOREIGN KEY (bucket_id) REFERENCES buckets(id);


--
-- Name: tnvnm-partition-name; Type: ACL; Schema: -; Owner: tnvnm
--

REVOKE ALL ON SCHEMA "tnvnm-partition-name" FROM PUBLIC;
REVOKE ALL ON SCHEMA "tnvnm-partition-name" FROM tnvnm;
GRANT ALL ON SCHEMA "tnvnm-partition-name" TO tnvnm;
GRANT USAGE ON SCHEMA "tnvnm-partition-name" TO "tnvnm-partition-name";
GRANT USAGE ON SCHEMA "tnvnm-partition-name" TO www;


--
-- Name: buckets; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE buckets FROM PUBLIC;
REVOKE ALL ON TABLE buckets FROM tnvnm;
GRANT ALL ON TABLE buckets TO tnvnm;
GRANT SELECT ON TABLE buckets TO www;
GRANT SELECT,UPDATE ON TABLE buckets TO "tnvnm-partition-name";


--
-- Name: buckets_id_seq; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON SEQUENCE buckets_id_seq FROM PUBLIC;
REVOKE ALL ON SEQUENCE buckets_id_seq FROM tnvnm;
GRANT ALL ON SEQUENCE buckets_id_seq TO tnvnm;
GRANT SELECT,UPDATE ON SEQUENCE buckets_id_seq TO "tnvnm-partition-name";


--
-- Name: events; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE events FROM PUBLIC;
REVOKE ALL ON TABLE events FROM tnvnm;
GRANT ALL ON TABLE events TO tnvnm;
GRANT SELECT ON TABLE events TO www;
GRANT SELECT,INSERT ON TABLE events TO "tnvnm-partition-name";


--
-- Name: events_id_seq; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON SEQUENCE events_id_seq FROM PUBLIC;
REVOKE ALL ON SEQUENCE events_id_seq FROM tnvnm;
GRANT ALL ON SEQUENCE events_id_seq TO tnvnm;
GRANT SELECT,UPDATE ON SEQUENCE events_id_seq TO "tnvnm-partition-name";


--
-- Name: filters; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE filters FROM PUBLIC;
REVOKE ALL ON TABLE filters FROM tnvnm;
GRANT ALL ON TABLE filters TO tnvnm;
GRANT SELECT,UPDATE ON TABLE filters TO "tnvnm-partition-name";


--
-- Name: filters_id_seq; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON SEQUENCE filters_id_seq FROM PUBLIC;
REVOKE ALL ON SEQUENCE filters_id_seq FROM tnvnm;
GRANT ALL ON SEQUENCE filters_id_seq TO tnvnm;
GRANT SELECT,UPDATE ON SEQUENCE filters_id_seq TO "tnvnm-partition-name";


--
-- Name: fingerprint_stats; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE fingerprint_stats FROM PUBLIC;
REVOKE ALL ON TABLE fingerprint_stats FROM tnvnm;
GRANT ALL ON TABLE fingerprint_stats TO tnvnm;
GRANT SELECT,INSERT,UPDATE ON TABLE fingerprint_stats TO "tnvnm-partition-name";


--
-- Name: fingerprints; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE fingerprints FROM PUBLIC;
REVOKE ALL ON TABLE fingerprints FROM tnvnm;
GRANT ALL ON TABLE fingerprints TO tnvnm;
GRANT SELECT,INSERT ON TABLE fingerprints TO "tnvnm-partition-name";


--
-- Name: fingerprints_id_seq; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON SEQUENCE fingerprints_id_seq FROM PUBLIC;
REVOKE ALL ON SEQUENCE fingerprints_id_seq FROM tnvnm;
GRANT ALL ON SEQUENCE fingerprints_id_seq TO tnvnm;
GRANT SELECT,UPDATE ON SEQUENCE fingerprints_id_seq TO "tnvnm-partition-name";


--
-- Name: ignored_hosts; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE ignored_hosts FROM PUBLIC;
REVOKE ALL ON TABLE ignored_hosts FROM tnvnm;
GRANT ALL ON TABLE ignored_hosts TO tnvnm;
GRANT SELECT ON TABLE ignored_hosts TO "tnvnm-partition-name";


--
-- Name: onlyon; Type: ACL; Schema: tnvnm-partition-name; Owner: tnvnm
--

REVOKE ALL ON TABLE onlyon FROM PUBLIC;
REVOKE ALL ON TABLE onlyon FROM tnvnm;
GRANT ALL ON TABLE onlyon TO tnvnm;
GRANT SELECT ON TABLE onlyon TO "tnvnm-partition-name";


--
-- PostgreSQL database dump complete
--