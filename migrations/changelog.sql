--liquibase formatted sql

--changeset david.salazar:1 
CREATE TABLE ticket (
    id INTEGER PRIMARY KEY AUTOINCREMENT not null,
    token TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    torn DATETIME
);
--rollback DROP TABLE ticket;

--changeset david.salazar:2
CREATE TABLE agent (
    id INTEGER PRIMARY KEY AUTOINCREMENT not null,
    hostname TEXT NOT NULL,
    ip TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    revoked DATETIME
)
--rollback DROP TABLE agent;
