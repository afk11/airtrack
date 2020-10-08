create table session (
    id serial not null primary key,
    identifier varchar(100) not null,
    project_id int not null,
    created_at timestamp null,
    updated_at timestamp null,
    closed_at timestamp null,
    deleted_at timestamp null,
    with_squawks bool not null,
    with_callsigns bool not null,
    with_transmission_types bool not null
);
create unique index unique_project_identifier on session(project_id, identifier);