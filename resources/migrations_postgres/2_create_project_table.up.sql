create table project (
    id serial not null primary key,
    identifier varchar(100) not null,
    label varchar(255) null,
    deleted_at timestamp null,
    created_at timestamp null,
    updated_at timestamp null);
create unique index project_identifier_unique on project(identifier);