create table collection_session (
    id serial not null primary key,
    identifier varchar(100) not null,
    project_id int not null,
    created_at timestamp null,
    updated_at timestamp null,
    closed_at timestamp null,
    deleted_at timestamp null);
create index collection_session_closed_at on collection_session(project_id);