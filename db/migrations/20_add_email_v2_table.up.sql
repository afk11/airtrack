create table `email_v2` (
    `id` int unsigned not null auto_increment primary key,
    `created_at` timestamp NOT NULL,
    `updated_at` timestamp NOT NULL,
    `retry_after` timestamp null,
    `status` int not null,
    `retries` int not null,
    `job` longblob not null
                        ) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `email_v2` add index `status`(`status`,`retry_after`);
