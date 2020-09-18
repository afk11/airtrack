create table email (
    id serial not null primary key,
    created_at timestamp null,
    updated_at timestamp null,
    retry_after timestamp null,
    status int not null,
    retries int not null,
    mail text not null);
create index email_status on email(status);