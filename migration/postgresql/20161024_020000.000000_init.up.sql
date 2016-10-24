CREATE TABLE user_status (
    id SERIAL,

    status VARCHAR(25) NOT NULL,

    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE "user" (
    id SERIAL,

    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    password CHAR(60) NOT NULL,

    status_id integer NOT NULL DEFAULT 1,

    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    UNIQUE (email),
    CONSTRAINT f_user_status FOREIGN KEY (status_id) REFERENCES user_status (id) ON DELETE CASCADE ON UPDATE CASCADE,

    PRIMARY KEY (id)
);

TRUNCATE TABLE user_status;

INSERT INTO user_status (id, status, created_at, updated_at, deleted_at) VALUES
(1, 'active',   CURRENT_TIMESTAMP,  NULL,  NULL),
(2, 'inactive', CURRENT_TIMESTAMP,  NULL,  NULL);

CREATE TABLE note (
    id SERIAL,

    name TEXT NOT NULL,

    user_id integer NOT NULL,

    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT NULL,
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    CONSTRAINT f_note_user FOREIGN KEY (user_id) REFERENCES "user" (id) ON DELETE CASCADE ON UPDATE CASCADE,

    PRIMARY KEY (id)
);

--GRANT SELECT,INSERT,UPDATE,DELETE ON "user" TO blueprint;
--GRANT SELECT,USAGE,UPDATE ON "user_id_seq" TO blueprint;
--GRANT SELECT,INSERT,UPDATE,DELETE ON user_status TO blueprint;
--GRANT SELECT,USAGE,UPDATE ON "user_status_id_seq" TO blueprint;
