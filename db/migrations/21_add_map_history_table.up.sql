create table `map_history` (
                            `id` int unsigned not null auto_increment primary key,
                            `project_id` bigint NOT NULL,
                            `time` timestamp NOT NULL,
                            `job` longblob not null
) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `map_history` add index `by_project`(`project_id`);
