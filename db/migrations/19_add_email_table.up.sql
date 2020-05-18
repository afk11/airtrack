create table `email` (`id` int unsigned not null auto_increment primary key, `created_at` timestamp null, `updated_at` timestamp null, `retry_after` timestamp null, `status` int not null, `retries` int not null, `mail` longtext not null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `email` add index `status`(`status`);