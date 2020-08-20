create table `email` (
    `id` integer not null primary key autoincrement,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `retry_after` timestamp null,
    `status` int not null,
    `retries` int not null,
    `mail` longtext not null
                     );
create index `email_status` on `email`(`status`);