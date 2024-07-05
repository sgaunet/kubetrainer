-- migrate:up

CREATE TABLE hits
(
    id SERIAL PRIMARY KEY,
    date timestamp NOT NULL
);

CREATE INDEX hits_id_idx       ON hits (id) ;

-- migrate:down

DROP TABLE hits cascade;
