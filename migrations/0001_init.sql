CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    login         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL CHECK (role IN ('viewer', 'organizer', 'admin'))
);

CREATE TABLE IF NOT EXISTS shows (
    id           BIGSERIAL PRIMARY KEY,
    organizer_id BIGINT NOT NULL REFERENCES users(id),
    title        VARCHAR(255) NOT NULL,
    venue        VARCHAR(255) NOT NULL,
    starts_at    TIMESTAMPTZ NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'draft'
                 CHECK (status IN ('draft', 'published', 'cancelled')),
    poster_path  VARCHAR(500)
);

CREATE TABLE IF NOT EXISTS seats (
    id      BIGSERIAL PRIMARY KEY,
    show_id BIGINT NOT NULL REFERENCES shows(id),
    row     INT NOT NULL,
    num     INT NOT NULL,
    price   BIGINT NOT NULL,
    status  VARCHAR(20) NOT NULL DEFAULT 'free'
            CHECK (status IN ('free', 'booked')),
    UNIQUE (show_id, row, num)
);

CREATE TABLE IF NOT EXISTS bookings (
    id         BIGSERIAL PRIMARY KEY,
    show_id    BIGINT NOT NULL REFERENCES shows(id),
    seat_id    BIGINT NOT NULL REFERENCES seats(id),
    user_id    BIGINT NOT NULL REFERENCES users(id),
    status     VARCHAR(20) NOT NULL DEFAULT 'booked'
               CHECK (status IN ('booked', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_seats_show_id ON seats(show_id);
CREATE INDEX IF NOT EXISTS idx_bookings_show_id ON bookings(show_id);
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
