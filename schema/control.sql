CREATE TABLE data_sources (
    id integer NOT NULL,
    name text NOT NULL,
    domains hstore NOT NULL,
    schema text NOT NULL
);
ALTER TABLE public.data_sources OWNER TO tnvnm;

CREATE SEQUENCE data_sources_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.data_sources_id_seq OWNER TO tnvnm;
ALTER SEQUENCE data_sources_id_seq OWNED BY data_sources.id;


ALTER TABLE ONLY data_sources ALTER COLUMN id SET DEFAULT nextval('data_sources_id_seq'::regclass);


ALTER TABLE ONLY data_sources
    ADD CONSTRAINT data_sources_domains_key UNIQUE (domains);

ALTER TABLE ONLY data_sources
    ADD CONSTRAINT data_sources_name_key UNIQUE (name);

ALTER TABLE ONLY data_sources
    ADD CONSTRAINT data_sources_pkey PRIMARY KEY (id);

ALTER TABLE ONLY data_sources
    ADD CONSTRAINT data_sources_server_key UNIQUE (schema);


REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM postgres;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO PUBLIC;


REVOKE ALL ON TABLE data_sources FROM PUBLIC;
REVOKE ALL ON TABLE data_sources FROM tnvnm;
GRANT ALL ON TABLE data_sources TO tnvnm;
GRANT SELECT ON TABLE data_sources TO www;