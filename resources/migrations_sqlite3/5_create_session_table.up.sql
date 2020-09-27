create table `session` (
    `id` integer not null primary key autoincrement,
    `identifier` varchar(100) not null,
    `project_id` int not null,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `closed_at` timestamp null,
    `deleted_at` timestamp null,
    `with_squawks` tinyint not null,
    `with_callsigns` tinyint not null,
    `with_transmission_types` tinyint not null
);

create index closed_at on session(`project_id`);