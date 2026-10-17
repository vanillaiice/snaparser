create table archive_source (
    id integer primary key default 1 check(id = 1),
    export_directory text not null
);
