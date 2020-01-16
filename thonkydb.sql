--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.13
-- Dumped by pg_dump version 9.6.13

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: battlefy; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.battlefy (
    team integer NOT NULL,
    stage_id character(24),
    team_id text,
    tournament_link text
);


ALTER TABLE public.battlefy OWNER TO pi;

--
-- Name: cache; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.cache (
    id character(44) NOT NULL,
    modified timestamp without time zone NOT NULL,
    players json NOT NULL,
    week json NOT NULL,
    activities text[] NOT NULL
);


ALTER TABLE public.cache OWNER TO pi;

--
-- Name: COLUMN cache.id; Type: COMMENT; Schema: public; Owner: pi
--

COMMENT ON COLUMN public.cache.id IS 'spreadsheet id';


--
-- Name: gamebattles; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.gamebattles (
    team integer NOT NULL,
    team_id text,
    tournament_link text
);


ALTER TABLE public.gamebattles OWNER TO pi;

--
-- Name: reminders; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.reminders (
    team integer NOT NULL,
    intervals integer[],
    activities text[],
    announce_channel text,
    role_mention text
);


ALTER TABLE public.reminders OWNER TO pi;

--
-- Name: schedules; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.schedules (
    team integer NOT NULL,
    spreadsheet_id text NOT NULL,
    update_interval integer NOT NULL
);


ALTER TABLE public.schedules OWNER TO pi;

--
-- Name: sheet_info; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.sheet_info (
    id character(44) NOT NULL,
    default_week json
);


ALTER TABLE public.sheet_info OWNER TO pi;

--
-- Name: teams; Type: TABLE; Schema: public; Owner: pi
--

CREATE TABLE public.teams (
    server_id bigint,
    team_name text,
    channels text[],
    id integer NOT NULL
);


ALTER TABLE public.teams OWNER TO pi;

--
-- Name: teams_id_seq; Type: SEQUENCE; Schema: public; Owner: pi
--

CREATE SEQUENCE public.teams_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.teams_id_seq OWNER TO pi;

--
-- Name: teams_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: pi
--

ALTER SEQUENCE public.teams_id_seq OWNED BY public.teams.id;


--
-- Name: teams id; Type: DEFAULT; Schema: public; Owner: pi
--

ALTER TABLE ONLY public.teams ALTER COLUMN id SET DEFAULT nextval('public.teams_id_seq'::regclass);


--
-- Name: battlefy battlefy_team_key; Type: CONSTRAINT; Schema: public; Owner: pi
--

ALTER TABLE ONLY public.battlefy
    ADD CONSTRAINT battlefy_team_key UNIQUE (team);


--
-- Name: cache cache_pkey; Type: CONSTRAINT; Schema: public; Owner: pi
--

ALTER TABLE ONLY public.cache
    ADD CONSTRAINT cache_pkey PRIMARY KEY (id);


--
-- Name: gamebattles gamebattles_team_key; Type: CONSTRAINT; Schema: public; Owner: pi
--

ALTER TABLE ONLY public.gamebattles
    ADD CONSTRAINT gamebattles_team_key UNIQUE (team);


--
-- Name: reminders reminders_team_key; Type: CONSTRAINT; Schema: public; Owner: pi
--

ALTER TABLE ONLY public.reminders
    ADD CONSTRAINT reminders_team_key UNIQUE (team);


--
-- Name: schedules schedules_team_key; Type: CONSTRAINT; Schema: public; Owner: pi
--

ALTER TABLE ONLY public.schedules
    ADD CONSTRAINT schedules_team_key UNIQUE (team);


--
-- PostgreSQL database dump complete
--

