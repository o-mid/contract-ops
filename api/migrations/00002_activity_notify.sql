-- +goose Up
-- Empty payloads on purpose: a notification is a wake-up, not the page.
-- Listeners re-read so a dropped or oversized message cannot publish a stale snapshot.
-- +goose StatementBegin
CREATE FUNCTION notify_activity_event() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify('activity_events', '');
  RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- One notice per statement, so seeding several fixtures does not fan out once per row.
CREATE TRIGGER activity_events_notify
AFTER INSERT OR UPDATE OR DELETE ON activity_events
FOR EACH STATEMENT
EXECUTE FUNCTION notify_activity_event();

-- +goose Down
DROP TRIGGER IF EXISTS activity_events_notify ON activity_events;
DROP FUNCTION IF EXISTS notify_activity_event();
