ALTER TABLE seats ADD COLUMN held_at TIMESTAMPTZ;

ALTER TABLE seats DROP CONSTRAINT IF EXISTS seats_status_check;
ALTER TABLE seats ADD CONSTRAINT seats_status_check
    CHECK (status IN ('free', 'held', 'booked'));