create table accounts (
    id integer primary key default 1 check(id = 1),
    username text not null,
    name text not null default '',
    creation_date text not null default '',
    registration_ip text not null default '',
    country text not null default '',
    last_active text not null default '',

    device_make text not null default '',
    device_model_id text not null default '',
    device_model_name text not null default '',
    device_language text not null default '',
    device_os_type text not null default '',
    device_os_version text not null default '',
    device_connection_type text not null default ''
);

create table device_history (
    id integer primary key,
    make text not null default '',
    model text not null default '',
    start_time text not null default '',
    device_type text not null default ''
);

create table login_history (
    id integer primary key,
    ip text not null default '',
    country text not null default '',
    created text not null default '',
    status text not null default '',
    device text not null default ''
);

-- combines the account-history arrays into one table.
create table account_changes (
    id integer primary key,
    change_type text not null,
    date text not null default '',
    status text not null default '',
    value text not null default ''
);

-- categories: friends, blocked, deleted, pending, etc.
create table friends (
    id integer primary key,
    username text not null,
    display_name text not null default '',
    creation_timestamp text not null default '',
    last_modified_timestamp text not null default '',
    source text not null default '',
    category text not null
);

create table conversations (
    id integer primary key,
    conversation_key text not null unique
);

create table chats (
    id integer primary key,
    conversation_id integer not null references conversations(id)
        on delete cascade,
    sender text not null default '',
    media_type text not null default '',
    created text not null default '',
    content text,
    conversation_title text,
    is_sender integer not null default 0
        check(is_sender in (0, 1)),
    created_microseconds integer not null default 0,
    is_saved integer not null default 0
        check(is_saved in (0, 1)),
    media_ids text not null default ''
);

create table snaps (
    id integer primary key,
    conversation_id integer not null references conversations(id)
        on delete cascade,
    sender text not null default '',
    media_type text not null default '',
    created text not null default '',
    conversation_title text,
    is_sender integer not null default 0
        check(is_sender in (0, 1)),
    created_microseconds integer not null default 0
);

-- type: incoming or outgoing
create table calls (
    id integer primary key,
    call_type text not null,
    date_time text not null default '',
    type text not null default '',
    people_in_chat integer not null default 0,
    result text not null default '',
    city text not null default '',
    country text not null default '',
    length_seconds integer not null default 0,
    network text not null default ''
);

create table snap_profiles (
    id integer primary key default 1 check(id = 1),
    created text not null default '',
    profile_title text not null default '',
    location text not null default '',
    profile_website text not null default '',
    profile_photo text not null default '',
    hero_image text not null default ''
);

create table user_profiles (
    id integer primary key default 1 check(id = 1),
    country text not null default '',
    creation_time text not null default '',
    account_creation_country text not null default '',
    platform_version text not null default '',
    in_app_language text not null default '',
    cohort_age text not null default '',
    derived_ad_demographic text not null default '',
    mobile_ad_id text not null default ''
);

create table engagements (
    id integer primary key,
    event text not null default '',
    occurrences integer not null default 0
);

create table discover_channels (
    id integer primary key,
    channel_name text not null default ''
);

create table ad_interactions (
    id integer primary key,
    advertiser_name text not null default '',
    date text not null default ''
);

create table user_profile_entries (
    id integer primary key,
    category text not null,
    value text not null default ''
);

create table import_history (
    id integer primary key,
    filename text not null,
    imported_at text not null default current_timestamp,
    inserted_count integer not null default 0
);

create index idx_friends_category
    on friends(category);

create index idx_friends_username
    on friends(username);

create index idx_chats_conversation_created
    on chats(conversation_id, created_microseconds);

create index idx_snaps_conversation_created
    on snaps(conversation_id, created_microseconds);

create index idx_user_profile_entries_category
    on user_profile_entries(category);
