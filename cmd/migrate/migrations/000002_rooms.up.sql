CREATE TABLE IF NOT EXISTS rooms(
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    author BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS roles(
    roleid INT PRIMARY KEY,
    name VARCHAR(10) NOT NULL UNIQUE,
    descript VARCHAR(100)
);

INSERT INTO roles (roleid, name, descript) 
VALUES (
    1,
    'guest',
    'create and comment'
),(
    2,
    'moderator',
    'guest perms and moderation'
),(
    3,
    'admin',
    'all access'
);

