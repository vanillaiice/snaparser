-- Keep source rows in their original tables and index searchable text with FTS5.
-- unicode61 supports multilingual text; remove_diacritics makes accented Latin
-- text searchable without accents. Use MATCH for words, phrases, and prefixes.

create virtual table chats_fts using fts5(
    content,
    sender,
    conversation_title,
    content='chats',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger chats_fts_ai after insert on chats begin
    insert into chats_fts(rowid, content, sender, conversation_title)
    values (new.id, new.content, new.sender, new.conversation_title);
end;

create trigger chats_fts_ad after delete on chats begin
    insert into chats_fts(chats_fts, rowid, content, sender, conversation_title)
    values ('delete', old.id, old.content, old.sender, old.conversation_title);
end;

create trigger chats_fts_au after update on chats begin
    insert into chats_fts(chats_fts, rowid, content, sender, conversation_title)
    values ('delete', old.id, old.content, old.sender, old.conversation_title);
    insert into chats_fts(rowid, content, sender, conversation_title)
    values (new.id, new.content, new.sender, new.conversation_title);
end;

-- Index rows imported before this migration.
insert into chats_fts(chats_fts) values ('rebuild');

create virtual table snaps_fts using fts5(
    sender,
    conversation_title,
    content='snaps',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger snaps_fts_ai after insert on snaps begin
    insert into snaps_fts(rowid, sender, conversation_title)
    values (new.id, new.sender, new.conversation_title);
end;

create trigger snaps_fts_ad after delete on snaps begin
    insert into snaps_fts(snaps_fts, rowid, sender, conversation_title)
    values ('delete', old.id, old.sender, old.conversation_title);
end;

create trigger snaps_fts_au after update on snaps begin
    insert into snaps_fts(snaps_fts, rowid, sender, conversation_title)
    values ('delete', old.id, old.sender, old.conversation_title);
    insert into snaps_fts(rowid, sender, conversation_title)
    values (new.id, new.sender, new.conversation_title);
end;

-- Index rows imported before this migration.
insert into snaps_fts(snaps_fts) values ('rebuild');

create virtual table friends_fts using fts5(
    username,
    display_name,
    content='friends',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger friends_fts_ai after insert on friends begin
    insert into friends_fts(rowid, username, display_name)
    values (new.id, new.username, new.display_name);
end;

create trigger friends_fts_ad after delete on friends begin
    insert into friends_fts(friends_fts, rowid, username, display_name)
    values ('delete', old.id, old.username, old.display_name);
end;

create trigger friends_fts_au after update on friends begin
    insert into friends_fts(friends_fts, rowid, username, display_name)
    values ('delete', old.id, old.username, old.display_name);
    insert into friends_fts(rowid, username, display_name)
    values (new.id, new.username, new.display_name);
end;

-- Index rows imported before this migration.
insert into friends_fts(friends_fts) values ('rebuild');

create virtual table accounts_fts using fts5(
    username,
    name,
    content='accounts',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger accounts_fts_ai after insert on accounts begin
    insert into accounts_fts(rowid, username, name)
    values (new.id, new.username, new.name);
end;

create trigger accounts_fts_ad after delete on accounts begin
    insert into accounts_fts(accounts_fts, rowid, username, name)
    values ('delete', old.id, old.username, old.name);
end;

create trigger accounts_fts_au after update on accounts begin
    insert into accounts_fts(accounts_fts, rowid, username, name)
    values ('delete', old.id, old.username, old.name);
    insert into accounts_fts(rowid, username, name)
    values (new.id, new.username, new.name);
end;

-- Index rows imported before this migration.
insert into accounts_fts(accounts_fts) values ('rebuild');

create virtual table snap_profiles_fts using fts5(
    profile_title,
    location,
    profile_website,
    content='snap_profiles',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger snap_profiles_fts_ai after insert on snap_profiles begin
    insert into snap_profiles_fts(rowid, profile_title, location, profile_website)
    values (new.id, new.profile_title, new.location, new.profile_website);
end;

create trigger snap_profiles_fts_ad after delete on snap_profiles begin
    insert into snap_profiles_fts(snap_profiles_fts, rowid, profile_title, location, profile_website)
    values ('delete', old.id, old.profile_title, old.location, old.profile_website);
end;

create trigger snap_profiles_fts_au after update on snap_profiles begin
    insert into snap_profiles_fts(snap_profiles_fts, rowid, profile_title, location, profile_website)
    values ('delete', old.id, old.profile_title, old.location, old.profile_website);
    insert into snap_profiles_fts(rowid, profile_title, location, profile_website)
    values (new.id, new.profile_title, new.location, new.profile_website);
end;

-- Index rows imported before this migration.
insert into snap_profiles_fts(snap_profiles_fts) values ('rebuild');

create virtual table discover_channels_fts using fts5(
    channel_name,
    content='discover_channels',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger discover_channels_fts_ai after insert on discover_channels begin
    insert into discover_channels_fts(rowid, channel_name)
    values (new.id, new.channel_name);
end;

create trigger discover_channels_fts_ad after delete on discover_channels begin
    insert into discover_channels_fts(discover_channels_fts, rowid, channel_name)
    values ('delete', old.id, old.channel_name);
end;

create trigger discover_channels_fts_au after update on discover_channels begin
    insert into discover_channels_fts(discover_channels_fts, rowid, channel_name)
    values ('delete', old.id, old.channel_name);
    insert into discover_channels_fts(rowid, channel_name)
    values (new.id, new.channel_name);
end;

-- Index rows imported before this migration.
insert into discover_channels_fts(discover_channels_fts) values ('rebuild');

create virtual table ad_interactions_fts using fts5(
    advertiser_name,
    content='ad_interactions',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger ad_interactions_fts_ai after insert on ad_interactions begin
    insert into ad_interactions_fts(rowid, advertiser_name)
    values (new.id, new.advertiser_name);
end;

create trigger ad_interactions_fts_ad after delete on ad_interactions begin
    insert into ad_interactions_fts(ad_interactions_fts, rowid, advertiser_name)
    values ('delete', old.id, old.advertiser_name);
end;

create trigger ad_interactions_fts_au after update on ad_interactions begin
    insert into ad_interactions_fts(ad_interactions_fts, rowid, advertiser_name)
    values ('delete', old.id, old.advertiser_name);
    insert into ad_interactions_fts(rowid, advertiser_name)
    values (new.id, new.advertiser_name);
end;

-- Index rows imported before this migration.
insert into ad_interactions_fts(ad_interactions_fts) values ('rebuild');

create virtual table user_profile_entries_fts using fts5(
    value,
    content='user_profile_entries',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

create trigger user_profile_entries_fts_ai after insert on user_profile_entries begin
    insert into user_profile_entries_fts(rowid, value)
    values (new.id, new.value);
end;

create trigger user_profile_entries_fts_ad after delete on user_profile_entries begin
    insert into user_profile_entries_fts(user_profile_entries_fts, rowid, value)
    values ('delete', old.id, old.value);
end;

create trigger user_profile_entries_fts_au after update on user_profile_entries begin
    insert into user_profile_entries_fts(user_profile_entries_fts, rowid, value)
    values ('delete', old.id, old.value);
    insert into user_profile_entries_fts(rowid, value)
    values (new.id, new.value);
end;

-- Index rows imported before this migration.
insert into user_profile_entries_fts(user_profile_entries_fts) values ('rebuild');
