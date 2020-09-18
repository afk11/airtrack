create table `email_v2` (
                            `id` integer not null primary key autoincrement,
                            `created_at` timestamp NOT NULL,
                            `updated_at` timestamp NOT NULL,
                            `retry_after` timestamp null,
                            `status` int not null,
                            `retries` int not null,
                            `job` longblob not null
);

create index `emailv2_status` on `email_v2`(`status`,`retry_after`);
