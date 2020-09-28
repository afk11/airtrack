create table `email` (
                         `id` integer not null primary key autoincrement,
                         `created_at` timestamp NOT NULL,
                         `updated_at` timestamp NOT NULL,
                         `retry_after` timestamp null,
                         `status` int not null,
                         `retries` int not null,
                         `job` longblob not null
                     );
create index `email_status` on `email`(`status`);