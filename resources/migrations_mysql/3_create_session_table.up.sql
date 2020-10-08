create table `session` (
    `id` int unsigned not null auto_increment primary key,
    `identifier` varchar(100) not null,
    `project_id` int not null,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `closed_at` timestamp null,
    `deleted_at` timestamp null,
    `with_squawks` tinyint not null,
    `with_callsigns` tinyint not null,
    `with_transmission_types` tinyint not null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `session` add unique index `unique_project_identifier`(`project_id`,`identifier`);