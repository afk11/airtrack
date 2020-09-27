create table `session` (
    `id` integer not null primary key autoincrement,
    `identifier` varchar(100) not null,
    `project_id` int not null,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `closed_at` timestamp null,
    `deleted_at` timestamp null
                                  );

create index closed_at on session(`project_id`);