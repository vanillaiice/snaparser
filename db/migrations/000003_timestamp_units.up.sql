-- Some exports label Unix milliseconds as "Created(microseconds)". Normalize
-- existing Snapchat-era timestamps to match the canonical microsecond columns.
-- Correct microsecond values and zero/unknown timestamps stay unchanged.
update chats
set created_microseconds = created_microseconds * 1000
where created_microseconds >= 1000000000000
  and created_microseconds < 100000000000000;

update snaps
set created_microseconds = created_microseconds * 1000
where created_microseconds >= 1000000000000
  and created_microseconds < 100000000000000;
