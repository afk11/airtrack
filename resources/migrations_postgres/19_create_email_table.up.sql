create table email (
                       id serial not null primary key,
                       created_at timestamp NOT NULL,
                       updated_at timestamp NOT NULL,
                       retry_after timestamp null,
                       status int not null,
                       retries int not null,
                       job bytea not null
    );
create index email_status on email(status);